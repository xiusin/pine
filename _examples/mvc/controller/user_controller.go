package controller

import (
	"github.com/xiusin/pine"
)

type UserController struct {
	pine.Controller
}

//func (user *UserController) GetIoc(
//	res http.ResponseWriter,
//	req *http.Request,
//	//sess sessions.ISession,
//	params *pine.Params,
//	cookie pine.ICookie,
//	rander *pine.Render,
//	cache cache.ICache) {
//	//cache.Save("name", []byte("xiusin"))
//	//re, _ := cache.Get("name")
//	//rander.Text(re)
//	//if !user.Ctx().ExistsCookie("login_name") {
//	//	user.Ctx().SetCookie("login_name", "xiusin", 10)
//	//}
//	//user.Ctx().Render().Text([]byte("hello world"))
//}

//func (user *UserController) GetLogin() interface{} {
//	index, _ := user.Ctx().GetInt("index", 0)
//	var p map[string]string
//	var m = map[int]interface{}{
//		0: "hello world",
//		1: 1,
//		2: 2.1,
//		3: struct {
//			Name string
//		}{Name: "xiusin"},
//		4: [5]int{1, 2, 3, 4, 5},
//		5: map[int]interface{}{
//			0: "hello world",
//			1: 1,
//			2: 2.1,
//			3: struct {
//				Name string
//			}{Name: "xiusin"},
//			4: [5]int{1, 2, 3, 4, 5},
//		},
//		6: true,
//		7: []interface{}{1,"hello", []int{1,2,3,4}, []byte("hello"),errors.New("hello"), nil, p},
//		10: &struct {
//			Name string
//		}{Name: "pointer"},
//		11: errors.New("发生了错误"),
//		12: nil,
//		13: p,
//		14: func() {},
//	}
//	d := m[index]
//	return d
//}

//func (user *UserController) GetLogin0() {
//}
//
//func (user *UserController) GetFlush() {
//	for range time.Tick(time.Second) {
//		user.Ctx().Flush("hello")
//	}
//}

