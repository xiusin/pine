package di

import "testing"

type test struct {
	Name string
}

func TestDefinition(t *testing.T) {
	build := NewBuilder()
	def := NewDefinition("test", func(builder BuilderInf) (i interface{}, e error) {
		t.Logf("callbacked")
		return &test{Name: "xiusin"}, nil
	}, true)
	build.Add(def)
	t.Logf("Resolved: %v, Shared: %v", def.IsResolved(), def.IsShared())
	s, err := def.Resolve(build)
	if err != nil {
		t.Error(err)
	}
	t.Logf("s.(*test).Name -> %s", s.(*test).Name)
	t.Logf("Resolved: %v", def.IsResolved())
	s1, err1 := build.Get("test")
	if err1 != nil {
		t.Error(err1)
	}
	t.Logf("s == s1 -> %v", s.(*test) == s1.(*test))
	// 将服务切换为非共享模式
	def1, err := build.GetDefinition("test")
	if err != nil {
		t.Error(err)
	}
	t.Logf("def == def1 -> %v", def == def1)
	def1.SetShared(false)
	s2, err1 := build.Get("test")

	if err1 != nil {
		t.Error(err1)
	}
	t.Logf("s1 == s2 -> %v", s1.(*test) == s2.(*test))
	build.Delete("test")

	_, err1 = build.Get("test")
	if err1 != nil {
		t.Logf("delete service : %s", err1.Error())
	}

	build.Set("test1", func(builder BuilderInf) (i interface{}, e error) {
		return &test{Name: "mirchen"}, nil
	}, true)
	s4, err1 := build.Get("test1")
	t.Logf("s4.(*test).Name -> %s", s4.(*test).Name)
	t.Logf("service exists: %v", build.Exists("test1"))
}
