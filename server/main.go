package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"log"
	"github.com/gorilla/handlers"
	"os"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"context"
	"fmt"
	//"io"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"time"
	"github.com/auth0/go-jwt-middleware"
)

type Project struct {
	Name      string `json:"name"`
}

const PORT string = ":8080"

func main(){
	r := mux.NewRouter()

	r.Handle("/containers/create", jwtMiddleware.Handler(CreateCon)).Methods("POST")
	r.Handle("/containers/{id}/start", jwtMiddleware.Handler(RunCon)).Methods("GET")
	r.Handle("/containers/{id}/stop", jwtMiddleware.Handler(StopCon)).Methods("GET")
	r.Handle("/containers/{id}/delete", jwtMiddleware.Handler(DeleteCon)).Methods("DELETE")
	r.Handle("/containers/{id}", jwtMiddleware.Handler(GetInfCon)).Methods("GET")
	r.Handle("/containers", jwtMiddleware.Handler(GetListCon)).Methods("GET")

	r.Handle("/get-token", GetTokenHandler).Methods("GET")
	log.Println("Server up and run on port " + PORT)
	log.Fatal(http.ListenAndServe(PORT, handlers.LoggingHandler(os.Stdout, r)))
}

var mySigningKey = []byte("secret")

var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return mySigningKey, nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})

var GetTokenHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
	// Создаем новый токен
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	
	// Устанавливаем набор параметров для токена
	claims["admin"] = true
	claims["name"] = "Ado Kukic"
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	// Подписываем токен нашим секретным ключем
	tokenString, _ := token.SignedString(mySigningKey)

	// Отдаем токен клиенту
	w.Write([]byte(tokenString))

})

var CreateCon = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	project := Project{}

	err := decoder.Decode(&project)
	if err != nil {
	log.Println(err)
	w.WriteHeader(http.StatusBadRequest)
	return
	}
	imageName := project.Name

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
	panic(err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
	Image: imageName,
	}, nil, nil, "")
	if err != nil {
	panic(err)
	}
	fmt.Println(resp.ID[:12])
	w.WriteHeader(http.StatusOK)
})

var RunCon = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/containers/"):len("/containers/") + 12]
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	if err := cli.ContainerStart(ctx, id, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
})

var StopCon = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/containers/"):len("/containers/") + 12]
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStop(ctx, id, nil);
		err != nil {
		panic(err)
	}

	fmt.Println("Success")
	w.WriteHeader(http.StatusOK)
})

var DeleteCon = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/containers/"):len("/containers/") + 12]
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{});
		err != nil {
		panic(err)
	}

	fmt.Println("Success")
	w.WriteHeader(http.StatusOK)
})

var GetInfCon = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/containers/"):len("/containers/") + 12]
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	inf, err := cli.ContainerInspect(ctx, id)
	if	err != nil {
		panic(err)
	}

	productsJson, _ := json.Marshal(inf)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(productsJson)
})

var GetListCon = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Println(container.ID[:12])
	}

	w.WriteHeader(http.StatusOK)
})


