package user

import (
	"gorm.io/gen"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
)

// Raw SQL
type Querier interface {
}

func RegisterUser(g *gen.Generator) {
	g.ApplyBasic(dbschema.User{})
	g.ApplyInterface(func(Querier) {}, dbschema.User{})
}
