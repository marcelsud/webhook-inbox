package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/eminetto/post-turso/book"
	"github.com/eminetto/post-turso/book/turso"
	"github.com/eminetto/post-turso/config"
	"github.com/eminetto/post-turso/internal/http/chi"
)

const TIMEOUT = 30 * time.Second

/* “a porta de entrada e saída da minha aplicação”
* Porque a porta de entrada? É no arquivo main.go, que vai ser compilado para gerar o executável da aplicação,
* onde é feita toda a “amarração” dos demais pacotes.
* É nele onde iniciamos as dependências, fazemos as configurações e a invocação dos pacotes que desempenham a lógica de negócio.

* E porque ele é a porta de saída da aplicação?
* https://eltonminetto.dev/post/2022-07-06-error-handling-cli-applications-golang/
 */

/*
 * As importações devem ser feitas apenas em uma direção: para baixo. O aplicativo (api, cli) importa camadas de negócios,
 * que importam a camada de armazenamento
 */

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
	)
	defer stop()
	repo, err := turso.NewRepository(cfg.DBName, cfg.TursoDatabaseURL, cfg.TursoAuthToken)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer repo.Close(ctx)
	s := book.NewService(repo)
	r := chi.Handlers(ctx, s)
	http.Handle("/", r)
	srv := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Addr:         ":" + cfg.Port,
		Handler:      http.DefaultServeMux,
	}

	errShutdown := make(chan error, 1)
	go shutdown(srv, ctx, errShutdown)
	fmt.Printf("Listening on port %s\n", cfg.Port)
	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
		return
	}
	err = <-errShutdown
	if err != nil {
		fmt.Println(err)
		return
	}
}

func shutdown(server *http.Server, ctxShutdown context.Context, errShutdown chan error) {
	<-ctxShutdown.Done()

	ctxTimeout, stop := context.WithTimeout(context.Background(), TIMEOUT)
	defer stop()

	err := server.Shutdown(ctxTimeout)
	switch err {
	case nil:
		fmt.Printf("\nShutting down server...\n")
		errShutdown <- nil
	case context.DeadlineExceeded:
		errShutdown <- fmt.Errorf("Forcing closing the server")
	default:
		errShutdown <- fmt.Errorf("Forcing closing the server")
	}
}
