package connector

import (
	"fmt"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector/database"
	"github.com/yaoapp/gou/connector/moapi"
	mongo "github.com/yaoapp/gou/connector/mongo"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/gou/connector/redis"
)

// Connectors the loaded connectors
var Connectors = map[string]Connector{}

// Load a connector from source
func Load(file string, id string) (Connector, error) {

	dsl := DSL{}
	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	err = application.Parse(file, data, &dsl)
	if err != nil {
		return nil, err
	}

	c, err := make(dsl.Type)
	if err != nil {
		return nil, err
	}

	err = c.Register(file, id, data)
	if err != nil {
		return nil, err
	}

	Connectors[id] = c
	return Connectors[id], nil
}

// New create a new connector
func New(typ string, id string, data []byte) (Connector, error) {
	c, err := make(typ)
	if err != nil {
		return nil, err
	}
	c.Register(id, "__source__", data)
	Connectors[id] = c
	return Connectors[id], nil
}

// Select a connector
func Select(id string) (Connector, error) {
	connector, has := Connectors[id]
	if !has {
		return nil, fmt.Errorf("connector %s not loaded", id)
	}
	return connector, nil
}

// Remove a connector
func Remove(id string) error {
	connector, has := Connectors[id]
	if !has {
		return fmt.Errorf("connector %s not loaded", id)
	}
	return connector.Close()
}

func make(typ string) (Connector, error) {

	t, has := types[typ]
	if !has {
		return nil, fmt.Errorf("%s does not support", typ)
	}

	switch t {
	case DATABASE:
		c := &database.Xun{}
		return c, nil

	case REDIS:
		c := &redis.Connector{}
		return c, nil

	case MONGO:
		c := &mongo.Connector{}
		return c, nil

	case OPENAI:
		c := &openai.Connector{}
		return c, nil

	case MOAPI:
		c := &moapi.Connector{}
		return c, nil
	}

	return nil, fmt.Errorf("%s does not support yet", typ)
}
