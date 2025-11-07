package controllers

import (
	"net/http"
)

type TestController struct{}

func StartTestController() (*TestController, error) {
	return &TestController{}, nil
}

func (c *TestController) GetTest(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("testandoooo.. testandoooooo..... um dois tressss...."))
}
