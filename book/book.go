package book

/* Sobre pacotes
 *
 * Os pacotes devem fornecer algo e não conter algo (ex: modelos, utilitários, auxiliares).
 * Isso (pacotes que contém algo) causa problemas de dependências, pois quando você precisa alterar algo,
 * precisa alterar muitos lugares
 *
 */

/*Sem tags, representa um livro em relação ao negócio */
// Define a book
type Book struct {
	ID       int64
	Title    string
	Author   string
	Category Category
}
