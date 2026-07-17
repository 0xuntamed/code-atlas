package httpapi

import "example.local/app/service"

func listAccounts() {
	service.LoadAccounts()
}

func Routes(router *gin.Engine) {
	router.GET("/accounts", authenticate, listAccounts)
}
