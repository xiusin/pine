package main

import (
	"fmt"
	"github.com/bxcodec/faker/v3"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/xiusin/pine"
	"github.com/xiusin/pine/di"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type ProductController struct {
	pine.Controller
}

func (p *ProductController) RegisterRoute(r pine.IRouterWrapper) {
	r.GET("/*action", "Handle")
}

// 控制器方法可以直接注入内置请求对象和di的参数对象
func (p *ProductController) Handle(orm *gorm.DB, params *pine.Params) interface{} {
	ormFromContext := p.Ctx().Value("orm").(*gorm.DB)
	fmt.Println(ormFromContext == orm) // true
	switch params.Get("action") {
	case "edit":
		id, _ := p.Ctx().GetInt64("id", 0)
		if id <= 0 {
			return "请输入要修改的数据ID"
		}
		p := &Product{
			Title: faker.FirstName() + " " + faker.LastName(),
			Price: uint(rand.Intn(1000)),
		}
		if orm.Model(p).Where("id = ?", id).Update(*p).RowsAffected > 0 {
			p.ID = uint(id)
			orm.First(p)
			return p
		}
		return "修改数据失败"
	case "delete-all":
		if orm.Where("1 = 1").Delete(&Product{}).RowsAffected > 0 {
			return true
		}
		return false
	case "delete":
		id, _ := p.Ctx().GetInt64("id", 0)
		if id <= 0 {
			return "请输入要删除的数据ID"
		}
		if orm.Where("id = ?", id).Delete(&Product{}).RowsAffected > 0 {
			return "删除成功"
		}
		return "删除失败"
	case "add":
		p := &Product{
			Title: faker.FirstName() + " " + faker.LastName(),
			Price: uint(rand.Intn(1000)),
		}
		orm.Create(p)
		if p.ID > 0 {
			return p
		}
		return "添加数据失败"
	default:
		var products []Product
		orm.Find(&products)
		return products
	}
}

type Product struct {
	gorm.Model
	Title string `json:"title"`
	Price uint   `json:"price"`
}


var db *gorm.DB

func init() {
	rand.Seed(time.Now().UnixNano())
	db, _ = gorm.Open("sqlite3", filepath.Join(os.TempDir(), "test.db"))
	db.AutoMigrate(&Product{})
	di.Set(&gorm.DB{}, func(builder di.BuilderInf) (i interface{}, err error) {
		return db, nil
	}, true)
	pine.RegisterOnInterrupt(func() {
		if err := db.Close(); err != nil {
			panic(err)
		}
	})
}

func main() {
	app := pine.New()
	app.Use(func(ctx *pine.Context) {
		ctx.Set("orm", db)
		ctx.Next()
	})
	app.Handle(new(ProductController))
	app.Run(pine.Addr(":9528"), pine.WithAutoParseControllerResult(true))
}
