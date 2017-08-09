package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"../app"
	"../model"
)

const (
	ok = "OK"
)

//Controller 响应控制器
type Controller struct {
}

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
	err := model.BindMemberReference(app.App.DB, id, ref)
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
	history, err := model.TransactionHistoryByID(app.App.DB, id, size, offset, greaterOrLess)
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}
	resp := historyResp{model.ResOK, ok, history}
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

type membersResp struct {
	RespCode string         `json:"respCode"`
	RespMsg  string         `json:"respMsg"`
	Members  []model.Member `json:"members"`
}

//CheckAccount 查询积分
//  id      : memberid
//  phone  : 消费金额 单位分, 例:120 = 1块2毛
//  cardno: 是否使用余额,缺省否
//  name: 姓名,姓名为关键字时,结果可能多个
//	至少1个不为空
func (c *Controller) CheckAccount(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := GetPara(r, "id")
	if len(id) == 0 {
		// err = m.FindByID(app.App.DB, id)
		// if err != nil {
		// 	fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		// 	return
		// }
		//} else {
		phone := GetPara(r, "phone")
		//fmt.Println(phone)
		cardno := GetPara(r, "cardno")
		name := GetPara(r, "name")
		members, code, msg := model.SearchMembersByInfo(app.App.DB, phone, cardno, name)
		if len(code) > 0 {
			if code != model.ResMore { //err
				fmt.Fprintf(w, JSONString(fillMemberMessageByCode(code, msg)))
				return
			}
			//else 多个用户结果
			fmt.Fprintf(w, JSONString(membersResp{model.ResMore, "请选择用户", members}))
			return
		}
		id = members[0].ID
	}
	//assert(m)
	//var d decimal.Decimal
	d, err := model.GetAmountByMember(app.App.DB, id)
	errMsg := &msgResp{}
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}
	resp := checkAccountResp{}
	resp.RespCode = model.ResOK
	resp.RespMsg = ok
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
	if err := m.FindByID(app.App.DB, id); err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}

	usePoint := GetPara(r, "usepoint")
	amount := GetPara(r, "amount")
	order := GetPara(r, "orderno")
	//fmt.Println("consume:", id, amount, usePoint)
	result, err := model.Consume(app.App.DB, m, amount, usePoint, order)
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

//AddUser 添加用户
//  phone     : 用户手机号
//  cardno    : 用户卡号,与手机号,至少一个非空. 不存在时, 创建新用户, 及其推荐返利关系树
//  name      : 用户名,老用户无效
//  refphone : 推荐人,识别为11位手机号 老用户无效
//  refcardno : 推荐人,卡号查询; 老用户无效
//  refname : 推荐人,姓名; 老用户无效(前两个为空,才使用)
//	refID	:	推荐人id,优先使用
//	return :
//		code = "200" 成功
//		code = "201" 用户已存在
//		code = "300" 推荐用户需要从多人中选择
//		code = "404" 引荐用户没找到
//		code = "412" 参数不足
//		code = "500" 内部错误
//		code = "501" 新用户创建失败
//
func (c *Controller) AddUser(w http.ResponseWriter, r *http.Request) {
	members, code := newUser(w, r)
	if code == model.ResDup {
		msg := userResp{}
		msg.CopyMemberInfo(&members[0])
		msg.RespCode = code
		msg.RespMsg = "用户已经存在"
		fmt.Fprintf(w, JSONString(msg))
	}
}

//CheckUser 检查用户
//  phone     : 用户手机号
//  cardno    : 用户卡号,与手机号,至少一个非空. 不存在时, 创建新用户, 及其推荐返利关系树
//  name      : 用户名,老用户无效
//  refphone : 推荐人,识别为11位手机号 老用户无效
//  refcardno : 推荐人,卡号查询; 老用户无效
//  refname : 推荐人,姓名; 老用户无效(前两个为空,才使用)
//	refID	:	推荐人id,优先使用
//	return :
//		code = "200" 成功
//		code = "300" 推荐用户需要从多人中选择
//		code = "404" 引荐用户没找到
//		code = "412" 参数不足
//		code = "500" 内部错误
//		code = "501" 新用户创建失败
//
func (c *Controller) CheckUser(w http.ResponseWriter, r *http.Request) {
	members, code := newUser(w, r)
	l := len(members)
	fmt.Println(code, l)
	if code == model.ResDup {
		//case model.ResFound: //老用户
		i, err1 := model.GetAmountByMember(app.App.DB, members[0].ID)
		if err1 != nil {
			code = model.ResFail
			fmt.Fprintf(w, (&(msgResp{})).messageString(code, err1.Error()))
			return
		}
		resp := userResp{}
		resp.Amount = i.String()
		resp.CopyMemberInfo(&members[0])

		resp.RespCode = code
		fmt.Fprintf(w, JSONString(resp))
	} //else 其他情况已返回
}

//Members 查找用户列表
//  id      : memberid
//  phone  : 消费金额 单位分, 例:120 = 1块2毛
//  cardno: 是否使用余额,缺省否
//  name: 姓名,姓名为关键字时,结果可能多个
//	至少1个不为空
func (c *Controller) Members(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := GetPara(r, "id")
	phone := GetPara(r, "phone")
	//fmt.Println(phone)
	cardno := GetPara(r, "cardno")
	name := GetPara(r, "name")

	members, code, msg := model.SearchMembers(app.App.DB, id, phone, cardno, name)

	if len(code) > 0 {
		fmt.Fprintf(w, JSONString(fillMemberMessageByCode(code, msg)))
		return
	}
	fmt.Fprintf(w, JSONString(membersResp{model.ResMore, "请选择用户", members}))
}

func fillMemberMessageByCode(code string, msg string) *msgResp {
	m := msgResp{RespCode: code}
	switch code {
	case model.ResInvalid:
		m.RespMsg = "请输入手机号或卡号或id,或姓名"
	case model.ResMore:
		m.RespMsg = "请选择用户"
		m.RespCode = model.ResOK
	case model.ResNotFound:
		m.RespMsg = "没有对应用户"
	case model.ResFail:
		m.RespMsg = msg
	default:
		if len(msg) == 0 {
			m.RespMsg = "未知错误"
		} else {
			m.RespMsg = msg
		}
	}
	return &m
}

//返回码,详见AddUser
func newUser(w http.ResponseWriter, r *http.Request) (members []model.Member, code string) {
	r.ParseForm() //解析参数，默认是不会解析的
	//  map :=
	resp := userResp{Amount: "0"}
	phone := GetPara(r, "phone")
	cardno := GetPara(r, "cardno")
	name := GetPara(r, "name")
	refID := GetPara(r, "refid")
	refphone := GetPara(r, "refphone")
	refcardno := GetPara(r, "refcardno")
	refname := GetPara(r, "refname")
	var m *model.Member
	//var members []model.Member
	var errstr string
	members, _, _ = model.SearchMembersByInfo(app.App.DB, phone, cardno, name)
	//fmt.Println("search result", members)
	if len(members) > 0 {
		for _, mem := range members {
			if model.NullStringEquals(mem.Phone, phone) || model.NullStringEquals(mem.CardNo, cardno) {
				code = model.ResDup
				//不输出结果, 其他方法需要自定义此场景下的返回//fmt.Fprintf(w, JSONString(msgResp{code, "用户已经存在"}))
				return
			}
		}
	}
	m, members, code, errstr = model.AddNewMember(app.App.DB, name, phone, cardno, refname, refphone, refcardno, refID, "")
	//fmt.Println(len(members), code, errstr, m)
	if code == model.ResMore1 {
		fmt.Fprintf(w, JSONString(membersResp{code, "请选择引荐用户", members}))
		return
	}
	if m == nil {
		fmt.Fprintf(w, JSONString(msgResp{code, errstr}))
		return
	}
	resp.CopyMemberInfo(m)
	resp.RespCode = code
	fmt.Fprintf(w, JSONString(resp))
	members = []model.Member{*m}
	return
}