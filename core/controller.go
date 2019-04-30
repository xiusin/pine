package core

type Controller struct {

}

func (c *Controller) Handle()  {

}

//b.Handle("GET", "/something/{id:long}", "MyCustomHandler", anyMiddleware...)


func (c *Controller)BeforeActivation(r *Router)  {

}







