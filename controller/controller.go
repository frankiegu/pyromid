//响应函数, 返回值详见../model/#pkg-constants
package controller

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	return jsonString(m)
}
func (m *msgResp) setMessage(code string, message string) {
	m.RespCode = code
	m.RespMsg = message
}

//getPara 获取key对应参数值; 不存在时,返回""
func getPara(r *http.Request, key string) string {
	arr := r.Form[key]
	if len(arr) > 0 {
		return arr[0]
	}
	return ""
}

//Bind 绑定推荐用户
//	  id     :被绑定会员id
//	  refid  :推荐会员id
//	  return :
//	    code = "200" 成功
//	    code = "412" 参数不足
//	    code = "500" 内部错误
func (c *Controller) Bind(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := getPara(r, "id")
	ref := getPara(r, "refid")
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

	fmt.Fprintf(w, errMsg.messageString(model.ResOK, ok))
}

type historyResp struct {
	RespCode string                     `json:"respCode"`
	RespMsg  string                     `json:"respMsg"`
	History  []model.HistoryTransaction `json:"history"`
}

//return nil when error
func stringToTime(s string) *time.Time {
	t, e := time.Parse("2006-1-2", s)
	if e != nil {
		fmt.Println(s, e.Error())
		return nil
	}
	return &t
}

//history 交易查询, 内部调用
//	  return :
//    code = "200" 成功
//    code = "300" 返回多位用户, 需要从多人中选择
//    code = "500" 内部错误
func (c *Controller) history(w http.ResponseWriter, r *http.Request, greaterOrLess string) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := getPara(r, "id")
	var name string
	//var err error
	errMsg := &msgResp{}
	if len(id) == 0 {
		members, code, msg := searchMember(r)
		//fmt.Println(code, msg, members)
		if code == model.ResMore {
			fmt.Fprintf(w, jsonString(membersResp{model.ResMore, "请选择用户", model.MapMembers2Output(members)}))
			return
		}
		if code != model.ResFound {
			fmt.Fprintf(w, errMsg.messageString(code, msg))
			return
		}
		id = members[0].ID
		name = members[0].Name.String
	}
	str := getPara(r, "pagesize")
	size, _ := strconv.Atoi(str)
	str = getPara(r, "offset")
	offset, _ := strconv.Atoi(str)
	str = getPara(r, "start")
	start := stringToTime(str)
	str = getPara(r, "end")
	end := stringToTime(str)
	//fmt.Println(str, end, start)
	//fmt.Println(id, size, offset)
	history, err := model.TransactionHistoryByID(app.App.DB, id, start, end, size, offset, greaterOrLess)
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}
	if len(name) == 0 {
		if len(history) > 0 {
			name = history[0].RelationName
		} else {
			m := model.NewMember()
			err := m.FindByID(app.App.DB, id)
			if err == nil {
				name = m.Name.String
			}
		}
	}
	resp := historyResp{model.ResOK, name, history}
	fmt.Fprintf(w, jsonString(resp))
}

//GainHistory 查询交易记录
//  id      : memberid
//  pagesize: optional
//  offset  : optional
//  start   : 2016-1-1
//  end     : 2016-1-2
//  return :
//    code = "200" 成功
//    code = "300" 返回多位用户, 需要从多人中选择
//    code = "500" 内部错误
func (c *Controller) GainHistory(w http.ResponseWriter, r *http.Request) {
	c.history(w, r, ">")
}

//ConsumeHistory 查询交易记录
//  id      : memberid
//  pagesize: optional
//  offset  : optional
//  start   : 2016-1-1
//  end     : 2016-1-2
//  return :
//    code = "200" 成功
//    code = "300" 返回多位用户, 需要从多人中选择
//    code = "500" 内部错误
func (c *Controller) ConsumeHistory(w http.ResponseWriter, r *http.Request) {
	c.history(w, r, "<")
}

type checkAccountResp struct {
	RespCode string `json:"respCode"`
	RespMsg  string `json:"respMsg"`
	Points   string `json:"points"`
}

type membersResp struct {
	RespCode string               `json:"respCode"`
	RespMsg  string               `json:"respMsg"`
	Members  []model.MemberOutput `json:"members"`
}

type referencesResp struct {
	RespCode string                  `json:"respCode"`
	RespMsg  string                  `json:"respMsg"`
	MemberID string                  `json:"id"`
	Name     string                  `json:"name"`
	Refs     []model.ReferenceOutput `json:"members"`
}

//CheckAccount 查询积分
//  id      : memberid
//  phone   : 用户手机号
//  cardno  : 是否使用余额,缺省否
//  name    : 姓名,姓名为关键字时,结果可能多个
//  至少1个不为空
//  return :
//    code = "200" 成功
//    code = "300" 返回多位用户, 需要从多人中选择
//    code = "500" 内部错误
func (c *Controller) CheckAccount(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := getPara(r, "id")
	if len(id) == 0 {
		// err = m.FindByID(app.App.DB, id)
		// if err != nil {
		// 	fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		// 	return
		// }
		//} else {
		phone := getPara(r, "phone")
		//fmt.Println(phone)
		cardno := getPara(r, "cardno")
		name := getPara(r, "name")
		members, code, msg := model.SearchMembersByInfo(app.App.DB, phone, cardno, name)
		if len(code) > 0 {
			if code != model.ResMore { //err
				fmt.Fprintf(w, jsonString(fillMemberMessageByCode(code, msg)))
				return
			}
			//else 多个用户结果
			fmt.Fprintf(w, jsonString(membersResp{model.ResMore, "请选择用户", model.MapMembers2Output(members)}))
			return
		}
		id = members[0].ID
	}
	//assert(m)
	//var d decimal.Decimal
	d, err := model.GetAmountByMember(app.App.DB, id, true)
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
	fmt.Fprintf(w, jsonString(resp))
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

//Cashout 提现
//  id      : memberid
//  amount  : 提现金额 单位分, 例:120 = 1块2毛
//  orderno : 订单号
//  return  :
//    code = "200" 成功
//    code = "201" 订单号重复, 订单号空时不做检查. 订单号与用户id联合检查, 相同用户id下,
//           订单号重复才失败;用户不同, 订单号相同,不算重复
//    code = "412" 余额不足
//    code = "500" 内部错误
func (c *Controller) Cashout(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := getPara(r, "id")
	if len(id) == 0 {
		fmt.Fprintf(w, jsonString(&msgResp{model.ResInvalid, "参数不足"}))
		return
	}
	m := model.NewMember()
	errMsg := &msgResp{}
	if err := m.FindByID(app.App.DB, id); err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}

	amount := getPara(r, "amount")
	order := getPara(r, "orderno")
	//fmt.Println("consume:", id, amount, usePoint)
	pointUsed, code, msg := model.Cashout(app.App.DB, m, amount, order)
	if code != model.ResOK {
		fmt.Fprintf(w, errMsg.messageString(code, msg))
	} else {
		resp := consumeResp{}
		resp.RespCode = model.ResOK
		resp.RespMsg = ok
		resp.MemberID = m.ID
		resp.PointUsed = pointUsed
		fmt.Fprintf(w, jsonString(resp))
	}
}

//Consume 消耗积分
//  id      : memberid
//  amount  : 消费金额 单位分, 例:120 = 1块2毛
//  usepoint: 是否使用余额,缺省否
//  orderno : 订单号
//  return  :
//    code = "200" 成功
//    code = "201" 订单号重复, 订单号空时不做检查. 订单号与用户id联合检查, 相同用户id下,
//           订单号重复才失败;用户不同, 订单号相同,不算重复
//    code = "500" 内部错误
func (c *Controller) Consume(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := getPara(r, "id")
	if len(id) == 0 {
		fmt.Fprintf(w, jsonString(&msgResp{model.ResInvalid, "参数不足"}))
		return
	}
	m := model.NewMember()
	errMsg := &msgResp{}
	if err := m.FindByID(app.App.DB, id); err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}

	usePoint := getPara(r, "usepoint")
	amount := getPara(r, "amount")
	order := getPara(r, "orderno")
	//fmt.Println("consume:", id, amount, usePoint)
	result, err := model.Consume(app.App.DB, m, amount, usePoint, order)
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
	} else {
		resp := consumeResp{}
		resp.RespCode = model.ResOK
		resp.RespMsg = ok
		resp.MemberID = m.ID
		resp.GainPoints = result.GainPoints
		resp.PayAmount = result.PayAmount
		resp.PointUsed = result.PointUsed
		resp.SelfGainPoints = result.SelfGainPoints
		fmt.Fprintf(w, jsonString(resp))
	}
}

type userResp struct {
	RespCode  string             `json:"respCode"`
	RespMsg   string             `json:"respMsg"`
	Member    model.MemberOutput `json:"member"`
	Reference model.MemberOutput `json:"reference"`
	Amount    string             `json:"amount"`
	Total     string             `json:"total"`
}

//jsonString output jason object
func jsonString(r interface{}) string {
	jb, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(jb)
}

func (r *userResp) CopyMemberInfo(m *model.Member, withAccount bool) {
	r.Member = *m.Map2Output()
	if m.Reference.Valid {
		ref := model.NewMember()
		if err := ref.FindByID(app.App.DB, m.Reference.String); err == nil {
			r.Reference = *ref.Map2Output()
		}
	}
	if withAccount {
		a, _ := model.GetAmountByMember(app.App.DB, m.ID, true)
		t, _ := model.GetAmountByMember(app.App.DB, m.ID, false)
		r.Amount = a.String()
		r.Total = t.Add(a).String()
	}
}

//UpdateUser 添加用户
//  phone  : 用户手机号
//  cardno : 用户卡号,原则上, 不需编辑卡号
//  name   : 用户名,与手机号至少一个不为空
//  id     : 用户id
//  return :
//    code = "200" 成功
//    code = "412" 参数不足
//    code = "500" 内部错误
func (c *Controller) UpdateUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	//  map :=
	id := getPara(r, "id")
	errMsg := &msgResp{}
	if len(id) == 0 {
		fmt.Fprintf(w, errMsg.messageString(model.ResInvalid, "id不能为空"))
		return
	}
	phone := getPara(r, "phone")
	cardno := getPara(r, "cardno")
	name := getPara(r, "name")
	if len(phone) == 0 || len(name) == 0 {
		fmt.Fprintf(w, errMsg.messageString(model.ResInvalid, "phone or name不能均为空"))
		return
	}
	err := model.UpdateMember(app.App.DB, id, phone, cardno, name)
	if err != nil {
		fmt.Fprintf(w, errMsg.messageString(model.ResFail, err.Error()))
		return
	}
	fmt.Fprintf(w, errMsg.messageString(model.ResOK, "更新成功"))
}

//AddUser 添加用户
//  phone     : 用户手机号
//  cardno    : 用户卡号,与手机号,至少一个非空. 不存在时, 创建新用户, 及其推荐返利关系树
//  name      : 用户名,老用户无效
//  refphone  : 推荐人,识别为11位手机号 老用户无效
//  refcardno : 推荐人,卡号查询; 老用户无效
//  refname   : 推荐人,姓名; 老用户无效(前两个为空,才使用)
//  refID     : 推荐人id,优先使用
//  return :
//    code = "200" 成功
//    code = "201" 用户已存在
//    code = "300" 推荐用户需要从多人中选择
//    code = "404" 引荐用户没找到
//    code = "412" 参数不足
//    code = "500" 内部错误
//    code = "501" 新用户创建失败
//
func (c *Controller) AddUser(w http.ResponseWriter, r *http.Request) {
	members, code := newUser(w, r)
	if code == model.ResDup {
		msg := userResp{RespCode: code, RespMsg: "用户已经存在"}
		msg.CopyMemberInfo(&members[0], false)
		fmt.Fprintf(w, jsonString(msg))
	}
}

//CheckUser 检查用户
//  phone     : 用户手机号
//  cardno    : 用户卡号,与手机号,至少一个非空. 不存在时, 创建新用户, 及其推荐返利关系树
//  name      : 用户名,老用户无效
//  refphone  : 推荐人,识别为11位手机号 老用户无效
//  refcardno : 推荐人,卡号查询; 老用户无效
//  refname   : 推荐人,姓名; 老用户无效(前两个为空,才使用)
//  refID     : 推荐人id,优先使用
//  return    :
//    code = "200" 成功
//    code = "300" 返回多位用户, 需要从多人中选择
//    code = "404" 引荐用户没找到
//    code = "412" 参数不足
//    code = "500" 内部错误
//    code = "501" 新用户创建失败
func (c *Controller) CheckUser(w http.ResponseWriter, r *http.Request) {
	members, code := newUser(w, r)
	//l := len(members)
	//fmt.Println(code, l)
	if code == model.ResDup {
		//case model.ResFound: //老用户
		i, err1 := model.GetAmountByMember(app.App.DB, members[0].ID, true)
		if err1 != nil {
			code = model.ResFail
			fmt.Fprintf(w, (&(msgResp{})).messageString(code, err1.Error()))
			return
		}
		resp := userResp{}
		resp.Amount = i.String()
		resp.CopyMemberInfo(&members[0], true)

		resp.RespCode = code
		fmt.Fprintf(w, jsonString(resp))
	} //else 其他情况已返回
}

//Reference 查找用户列表
//  id    : 推荐人id
//  phone : 消费金额 单位分, 例:120 = 1块2毛
//  cardno: 是否使用余额,缺省否
//  name  : 姓名,姓名为关键字时,结果可能多个
//	至少1个不为空
//  return:
//    code = "200" 成功
//    code = "300" 推荐用户需要从多人中选择
//    code = "500" 内部错误
func (c *Controller) Reference(w http.ResponseWriter, r *http.Request) {
	members, code, msg := searchMember(r)
	//fmt.Println("ref back", members, code, msg)
	if code == model.ResMore {
		fmt.Fprintf(w, jsonString(membersResp{model.ResMore, "请选择用户", model.MapMembers2Output(members)}))
		return
	}
	if code == model.ResFound {
		m := members[0]
		refs, err := model.FindReferenceByID(app.App.DB, m.ID)
		//fmt.Println("ref result", m, members, err)
		if err != nil {
			if sql.ErrNoRows == err {
				fmt.Fprintf(w, jsonString(getMsgRespByCode(model.ResNotFound)))
				return
			}
			code = model.ResFail
			msg = err.Error()
		} else {
			fmt.Fprintf(w, jsonString(referencesResp{model.ResMore1, m.Name.String, m.ID, m.Name.String, model.MapReference2Output(refs)}))
			return
		}
	}
	fmt.Fprintf(w, jsonString(fillMemberMessageByCode(code, msg)))
}

//Members 查找用户列表
//  id    : memberid
//  phone : 电话
//  cardno: 是否使用余额,缺省否
//  name  : 姓名,姓名为关键字时,结果可能多个
//  至少1个不为空
//  return:
//    code = "200" 成功
//    code = "300" 返回多位用户, 需要从多人中选择
//    code = "500" 内部错误
func (c *Controller) Members(w http.ResponseWriter, r *http.Request) {
	members, code, msg := searchMember(r)
	//fmt.Println(code, msg, members)
	if code == model.ResMore {
		fmt.Fprintf(w, jsonString(membersResp{model.ResMore, "请选择用户", model.MapMembers2Output(members)}))
		return
	}
	if code == model.ResFound {
		u := userResp{RespCode: model.ResFound, RespMsg: msg}
		u.CopyMemberInfo(&members[0], true)
		fmt.Fprintf(w, jsonString(u))
		return
	}
	fmt.Fprintf(w, jsonString(fillMemberMessageByCode(code, msg)))
}

//GetRatio 获取当前分成比例
func (c *Controller) GetRatio(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, model.GetRatioJSON())
}

//SetRatio 设置当前分成比例
//	ratio :
//	syncall : bool是否更新已有记录
//	updateall : bool更新是否检查与现有ratio相同, true 所有更新, false只更新与当前ratio相同的
func (c *Controller) SetRatio(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ratios := r.Form["ratio"]
	fmt.Println(ratios)
	sync := getPara(r, "syncall")
	updAll := getPara(r, "updateall")

	code, msg := model.UpdateRatios(app.App.DB, ratios, sync, updAll)
	fmt.Fprintf(w, jsonString(&msgResp{code, msg}))
}

func getMsgRespByCode(code string) *msgResp {
	var msg string
	switch code {
	case model.ResInvalid:
		msg = "请输入手机号或卡号或id,或姓名"
	case model.ResMore:
		msg = "请选择用户"
		//m.RespCode = model.ResOK
	case model.ResNotFound:
		msg = "没有对应用户"
	case model.ResFail: //do nothing
		//msg = msg
	case model.ResFound:
		msg = ok
	case model.ResPhoneInvalid:
		msg = "无效手机号"
	default:
		if len(msg) == 0 {
			msg = "未知错误" + code
			//panic(msg)
		} else { //do nothing
			//			msg = msg
		}
	}
	return &msgResp{code, msg}
}

func fillMemberMessageByCode(code string, msg string) *msgResp {
	m := getMsgRespByCode(code)
	if len(m.RespMsg) == 0 {
		m.RespMsg = msg
	}
	return m
}

//返回码,详见Members model.SearchMembersByInfo
func searchMember(r *http.Request) ([]model.Member, string, string) {
	r.ParseForm() //解析参数，默认是不会解析的
	id := getPara(r, "id")
	phone := getPara(r, "phone")
	//fmt.Println(phone)
	cardno := getPara(r, "cardno")
	name := getPara(r, "name")

	return model.SearchMembers(app.App.DB, id, phone, cardno, name)
}

//返回码,详见AddUser
func newUser(w http.ResponseWriter, r *http.Request) (members []model.Member, code string) {
	r.ParseForm() //解析参数，默认是不会解析的
	//  map :=
	resp := userResp{Amount: "0"}
	phone := getPara(r, "phone")
	cardno := getPara(r, "cardno")
	name := getPara(r, "name")
	refID := getPara(r, "refid")
	refphone := getPara(r, "refphone")
	refcardno := getPara(r, "refcardno")
	refname := getPara(r, "refname")
	var m *model.Member
	//var members []model.Member
	var errstr string
	members, _, _ = model.SearchMembersByInfo(app.App.DB, phone, cardno, name)
	//fmt.Println("search result", members)
	if len(members) > 0 {
		for _, mem := range members {
			if model.NullStringEquals(mem.Phone, phone) || model.NullStringEquals(mem.CardNo, cardno) {
				code = model.ResDup
				//不输出结果, 其他方法需要自定义此场景下的返回//fmt.Fprintf(w, jsonString(msgResp{code, "用户已经存在"}))
				return
			}
		}
	}
	m, members, code, errstr = model.AddNewMember(app.App.DB, name, phone, cardno, refname, refphone, refcardno, refID, "")
	//fmt.Println(len(members), code, errstr, m)
	if code == model.ResMore1 {
		fmt.Fprintf(w, jsonString(membersResp{code, "请选择引荐用户", model.MapMembers2Output(members)}))
		return
	}
	if m == nil {
		fmt.Fprintf(w, jsonString(msgResp{code, errstr}))
		return
	}
	resp.CopyMemberInfo(m, false)
	resp.RespCode = code
	fmt.Fprintf(w, jsonString(resp))
	members = []model.Member{*m}
	return
}
