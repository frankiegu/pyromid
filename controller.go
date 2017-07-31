package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	_ "strings"

	"github.com/shopspring/decimal"

	"./model"
)

type msgResp struct {
	RespCode string `json:"respCode"`
	RespMsg  string `json:"respMsg"`
}

func (m *msgResp) messageString(code string, message string) string {
	m.setMessage(code, message)
	return JSONString(m)
}
func (m *msgResp) setMessage(code string, message string) {
	m.RespCode = code
	m.RespMsg = message
}

//GetPara 获取key对应参数值, 不存在返回""
func GetPara(r *http.Request, key string) string {
	arr := r.Form[key]
	if len(arr) > 0 {
		return arr[0]
	}
	return ""
}

//Bind 绑定用户推荐
//	id 被绑定会员id
//	ref 推荐会员id
func (c *Controller) Bind(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := GetPara(r, "id")
	ref := GetPara(r, "ref")
	errMsg := &msgResp{}
	if len(id) == 0 || len(ref) == 0 {
		fmt.Fprintf(w, errMsg.messageString(model.ResInvalid, "id or ref不能为空"))
		return
	}
	err := model.BindMemberReference(App.DB, id, ref)
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}

	fmt.Fprintf(w, errMsg.messageString(model.ResOK, "Done"))
}

type historyResp struct {
	RespCode string                     `json:"respCode"`
	RespMsg  string                     `json:"respMsg"`
	History  []model.HistoryTransaction `json:"history"`
}

func (c *Controller) history(w http.ResponseWriter, r *http.Request, greaterOrLess string) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := GetPara(r, "id")
	//var err error
	errMsg := &msgResp{}
	if len(id) == 0 {
		fmt.Fprintf(w, errMsg.messageString(model.ResInvalid, "id不能为空"))
		return
	}
	str := GetPara(r, "pagesize")
	size, _ := strconv.Atoi(str)
	str = GetPara(r, "offset")
	offset, _ := strconv.Atoi(str)
	//fmt.Println(id, size, offset)
	history, err := model.TransactionHistoryByID(App.DB, id, size, offset, greaterOrLess)
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}
	resp := historyResp{model.ResOK, "OK", history}
	fmt.Fprintf(w, JSONString(resp))
}

//GainHistory 查询交易记录
//  id      : memberid
//	pagesize:
//	offset:
func (c *Controller) GainHistory(w http.ResponseWriter, r *http.Request) {
	c.history(w, r, ">")
}

//ConsumeHistory 查询交易记录
//  id      : memberid
//	pagesize:
//	offset:
func (c *Controller) ConsumeHistory(w http.ResponseWriter, r *http.Request) {
	c.history(w, r, "<")
}

type checkAccountResp struct {
	RespCode string `json:"respCode"`
	RespMsg  string `json:"respMsg"`
	Points   string `json:"points"`
}

//CheckAccount 查询积分
//  id      : memberid
//  phone  : 消费金额 单位分, 例:120 = 1块2毛
//  cardno: 是否使用余额,缺省否
//	至少1个不为空
func (c *Controller) CheckAccount(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := GetPara(r, "id")
	var m *model.Member
	var err error
	m = model.NewMember()
	errMsg := &msgResp{}
	if len(id) != 0 {
		err = m.FindByID(App.DB, id)
		if err != nil {
			fmt.Fprintf(w, errMsg.messageString(model.ResInvalid, err.Error()))
			return
		}
	} else {
		phone := GetPara(r, "phone")
		//fmt.Println(phone)
		cardno := GetPara(r, "cardno")
		if len(phone) == 0 && len(cardno) == 0 {
			fmt.Fprintf(w, errMsg.messageString(model.ResInvalid, "请输入手机号或卡号或id"))
			return
		}
		_, err = m.FindByPhoneOrCardno(App.DB, phone, cardno)
		if err != nil {
			fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
			return
		}
	}
	//assert(m)
	var d decimal.Decimal
	d, err = model.GetAmountByMember(App.DB, m.ID)
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}
	resp := checkAccountResp{}
	resp.RespCode = model.ResOK
	resp.RespMsg = "OK"
	resp.Points = d.String()
	//fmt.Println("ck account:", resp)
	fmt.Fprintf(w, JSONString(resp))
}

type consumeResp struct {
	RespCode       string `json:"respCode"`
	RespMsg        string `json:"respMsg"`
	MemberID       string `json:"id"`
	PointUsed      string `json:"pointused"`
	PayAmount      string `json:"payamount"`
	SelfGainPoints string `json:"selfgainpoints"`
	GainPoints     string `json:"gainpoints"`
}

//Consume 消耗积分
//  id      : memberid
//  amount  : 消费金额 单位分, 例:120 = 1块2毛
//  usepoint: 是否使用余额,缺省否
//	orderno	:	订单号
func (c *Controller) Consume(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := GetPara(r, "id")
	m := model.NewMember()
	errMsg := &msgResp{}
	if err := m.FindByID(App.DB, id); err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}

	usePoint := GetPara(r, "usepoint")
	amount := GetPara(r, "amount")
	order := GetPara(r, "orderno")
	//fmt.Println("consume:", id, amount, usePoint)
	result, err := model.Consume(App.DB, m, amount, usePoint, order)
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
	} else {
		resp := consumeResp{}
		resp.RespCode = model.ResOK
		resp.RespMsg = "ok"
		resp.MemberID = m.ID
		resp.GainPoints = result.GainPoints
		resp.PayAmount = result.PayAmount
		resp.PointUsed = result.PointUsed
		resp.SelfGainPoints = result.SelfGainPoints
		fmt.Fprintf(w, JSONString(resp))
	}
}

type userResp struct {
	RespCode string `json:"respCode"`
	RespMsg  string `json:"respMsg"`
	MemberID string `json:"id"`
	Amount   string `json:"amount"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}

//JSONString output jason object
func JSONString(r interface{}) string {
	jb, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(jb)
}
func (r *userResp) CopyMemberInfo(m *model.Member) {
	r.MemberID = m.ID
	r.Name = m.Name.String
	r.Phone = m.Phone.String
}

//CheckUser 检查用户
//  phone     : 用户手机号
//  cardno    : 用户卡号,与手机号,至少一个非空. 不存在时, 创建新用户, 及其推荐返利关系树
//  reference : 推荐人,识别为11位手机号,按手机号,否则按卡号查询; 老用户无效
//  Name      : 用户名,老用户无效
func (c *Controller) CheckUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	//  map :=
	resp := userResp{Amount: "0"}
	phone := GetPara(r, "phone")
	//fmt.Println(phone)
	cardno := GetPara(r, "cardno")
	m := model.NewMember()
	code, err := m.FindByPhoneOrCardno(App.DB, phone, cardno)
	//fmt.Println(err,code)
	switch code {
	case model.ResNotFound: //需建新用户
		name := GetPara(r, "name")
		reference := GetPara(r, "reference")
		m = model.AddNewMember(App.DB, phone, cardno, reference, "", name)
		if m == nil {
			err = errors.New("用户创建失败" + phone)
			code = model.ResFailCreateMember
		} else {
			err = nil
			code = model.ResOK
			resp.CopyMemberInfo(m)
		}
	case model.ResFound: //老用户
		i, err1 := model.GetAmountByMember(App.DB, m.ID)
		if err1 != nil {
			code = model.ResFail
		} else {
			resp.Amount = i.String()
			resp.CopyMemberInfo(m)
		}
	}
	if err != nil { //其他错误
		errMsg := &msgResp{}
		fmt.Fprintf(w, errMsg.messageString(code, err.Error()))
		return
	}
	resp.RespCode = code
	fmt.Fprintf(w, JSONString(resp))
}
