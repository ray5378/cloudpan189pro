package context

import "github.com/bwmarrin/snowflake"

var snowflakeNode *snowflake.Node

func init() {
	snowflakeNode, _ = snowflake.NewNode(1)
}

func generateUniqueID() string {
	return snowflakeNode.Generate().String()
}
