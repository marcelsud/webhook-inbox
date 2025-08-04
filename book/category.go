package book

import "bytes"

/* Criar tipos de dados específicos para a aplicação
* Usar o compilador a seu favor, tentar encontrar erros em tempo de compilação e não de execução.
 */

// Category type
type Category int

const (
	WantToRead Category = iota + 1
	Reading
	Read
)

func (c Category) String() string {
	switch c {
	case WantToRead:
		return "Want to Read"
	case Read:
		return "Read"
	case Reading:
		return "Reading"
	}
	return "Unknown"
}

/* Define how to transform a Category object into a JSON. Example of using the standard language interfaces
 * https://eltonminetto.dev/post/2022-06-07-using-go-interfaces/
 */

// Define how to transform a Category object into a JSON
func (c Category) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(c.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

/*
 * Sobre o receiver das funções ser ponteiro ou valor:
 * se precisa mudar o valor usa-se ponteiro, se não usa valor.
 * Usar valor é mais performático pq cada instância tem seu valor e não precisa alocar na heap,
 * reduzindo custo de GC
 */

// Create a new category
func NewCategory(s string) Category {
	switch s {
	case "Want to Read":
		return WantToRead
	case "Read":
		return Read
	case "Reading":
		return Reading
	}
	return WantToRead
}
