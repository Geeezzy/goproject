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
	"encoding/json"
	"encoding/base64"
	"github.com/dgrijalva/jwt-go"
	"time"
	"github.com/auth0/go-jwt-middleware"
	"strings"
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
	r.Handle("/get-token", GetTokenHandler).Methods("POST")

	log.Println("Server up and run on port " + PORT)
	log.Fatal(http.ListenAndServe(PORT, handlers.LoggingHandler(os.Stdout, r)))
}

// Глобальный секретный ключ
var mySigningKey = []byte("secret")

//Прослойка, которая будет выполнять проверку нашего токена
var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return mySigningKey, nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})

//Роутер, который генерирует новый JWT и аутентификация пользователя
var GetTokenHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){

	auth := strings.SplitN(r.Header["Authorization"][0], " ", 2)
	if len(auth) != 2 || auth[0] != "Basic" {
		http.Error(w, "bad syntax", http.StatusBadRequest)
		return
	}

	payload, _ := base64.StdEncoding.DecodeString(auth[1])
	pair := strings.SplitN(string(payload), ":", 2)

	if Validate(pair[0], pair[1]) {
		// Создаем новый токен
		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)

		// Устанавливаем набор параметров для токена
		claims["admin"] = true
		claims["name"] = "klyov"
		claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

		// Подписываем токен нашим секретным ключем
		tokenString, err := token.SignedString(mySigningKey)
		if err != nil {
			panic(err)
		}
		// Отдаем токен клиенту
		w.Write([]byte(tokenString))
		w.WriteHeader(http.StatusOK)
	} else{
		w.WriteHeader(http.StatusUnauthorized)
	}

})

//Роутер, создающий контейнер
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

//Роутер, запускающий контейнер
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

//Роутер, останавливающий контейнер
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

//Роутер, удаляющий контейнер
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

//Роутер, который выдает информацию по контейнеру
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

	productsJson, err := json.Marshal(inf)
	if	err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(productsJson)
})

//Роутер, который выдает список запущенных контейнеров
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

//Проверка введеных данных
func Validate(username, password string) bool {
	if username == "Geeezzy" && password == "Trapa35" {
		return true
	}
	return false
}
