package book

import "context"

/* Interfaces pequenas */

/* Interfaces abstraem comportamento e não coisas*/

/* Não escrevemos interfaces para testes, mas sim para usuários (que utilizam a api/interface) */

type Reader interface {
	/* Funções fazem data transformation. E devem ter apenas um propósito. O nome da função deve representar isso. Devemos testar isso */
	/* Funções tem como seu primeiro parametro um Context e retornam um error */
	Select(ctx context.Context, id int64) (Book, error)
	SelectAll(ctx context.Context) ([]Book, error)
}

type Writer interface {
	Insert(ctx context.Context, book Book) (int64, error)
	Update(ctx context.Context, book Book) error
	Delete(ctx context.Context, id int64) error
}

/* Composição de interfaces */

type Repository interface {
	Reader
	Writer
	Close(ctx context.Context) error
}
