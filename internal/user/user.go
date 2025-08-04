package user

/*
* Usar o pacote de negógio user dentro de internal garante que ele vai estar protegido pois arquivos dentro de um
* diretório `internal` só podem ser importados por
* pacotes que estejam em diretórios ancestrais (pais) ao diretório `internal`.
* Isso gera uma barreira efetiva para que códigos de fora daquele módulo ou projeto não consigam
* acessar o conteúdo interno.
* Por exemplo, um pacote em `github.com/exemplo/projeto/internal` só poderá ser importado por
* pacotes dentro de `github.com/exemplo/projeto` e seus subdiretórios, não por pacotes externos
 */
