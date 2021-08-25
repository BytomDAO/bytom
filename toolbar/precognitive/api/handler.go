package api

import (
	"github.com/gin-gonic/gin"

	"github.com/bytom/bytom/toolbar/precognitive/database/orm"
	serverCommon "github.com/bytom/bytom/toolbar/server"
)

type listNodesReq struct{ serverCommon.Display }

func (s *Server) ListNodes(c *gin.Context, listNodesReq *listNodesReq, query *serverCommon.PaginationQuery) ([]*orm.Node, error) {
	var ormNodes []*orm.Node
	if err := s.db.Offset(query.Start).Limit(query.Limit).Find(&ormNodes).Error; err != nil {
		return nil, err
	}

	return ormNodes, nil
}
