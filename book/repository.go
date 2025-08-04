package book

import "context"

/* Interfaces pequenas */

/*
 * Quando uma struct representa DADOS deveria usar sempre value semantics e não pointer (ex: Book) .
 * Se a struct representa uma API deveria ser pointer (ex: Service).
 * Para tipos primários (int, string) sempre value semantics
 * Para tipos internos (maps, slices) usar value semantics
 */

/* Funções fazem data transformation. E devem ter apenas um propósito. O nome da função deve representar isso. Devemos testar isso */

/* Interfaces abstraem comportamento e não coisas*/

/* Não escrevemos interfaces para testes, mas sim para usuários (que utilizam a api/interface) */

type Reader interface {
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
