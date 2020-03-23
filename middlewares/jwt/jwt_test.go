package jwt

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/xiusin/pine"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

const JWTSignKey = "AllYourBase"

var SingnedMethod = jwt.SigningMethodHS256

func TestJwt(t *testing.T) {
	app := pine.New()
	jwtM := NewJwt(JwtOptions{
		Secret:        []byte(JWTSignKey),
		SigningMethod: SingnedMethod,
		Debug:         true,
	})

	app.GET("/hello/:name", func(c *pine.Context) {
		chaim := c.Value("jwt.tokenClaims").(*jwt.StandardClaims)
		_, _ = c.Writer().Write([]byte("jwt " + chaim.Issuer + " Hello " + c.Params().Get("name")))
	}, jwtM.Serve())


	app.GET("/get/jwt", func(context *pine.Context) {
		mySigningKey := []byte(JWTSignKey)
		claims := &jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Second * 10000).Unix(),
			Issuer:    "test",
		}
		token := jwt.NewWithClaims(SingnedMethod, claims)
		ss, err := token.SignedString(mySigningKey)
		if err != nil {
			panic(err)
		}
		if _, err = context.Writer().Write([]byte(ss)); err != nil {
			panic(err)
		}
	})
	srv := httptest.NewServer(app)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/get/jwt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("获取token", string(content))
	req, err := http.NewRequest("GET", srv.URL+"/hello/xiusin", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", string(content)))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatal("状态码错误, 期待200得到" + strconv.Itoa(res.StatusCode))
	}
	content, _ = ioutil.ReadAll(res.Body)
	if string(content) == "jwt test Hello xiusin" {
		t.Log("测试token成功")
	} else {
		t.Fatal("测试token失败")
	}
}
