// Package docs provides the Swagger documentation
package docs

import "github.com/swaggo/swag"

var doc = `{
    "swagger": "2.0",
    "info": {
        "title": "BetterMode API Scraper",
        "description": "A server that retrieves content from BetterMode API",
        "version": "1.0"
    },
    "host": "gpters.automationpro.online",
    "basePath": "/api/v1",
    "schemes": ["https"],
    "paths": {
        "/content": {
            "post": {
                "description": "Retrieves content value from mappingFields where key is \"content\"",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["content"],
                "summary": "Get content from BetterMode API",
                "parameters": [
                    {
                        "description": "Post ID and optional format",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "post_id": {
                                    "type": "string",
                                    "description": "The BetterMode post ID to retrieve"
                                },
                                "format": {
                                    "type": "string",
                                    "description": "Format of the returned content",
                                    "enum": ["html", "text"],
                                    "default": "html"
                                }
                            },
                            "required": ["post_id"]
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "content": {
                                    "type": "string",
                                    "description": "The content of the post"
                                },
                                "format": {
                                    "type": "string",
                                    "description": "The format of the content (html or text)"
                                },
                                "post_id": {
                                    "type": "string",
                                    "description": "The ID of the post"
                                },
                                "title": {
                                    "type": "string",
                                    "description": "The title of the post"
                                },
                                "char_count": {
                                    "type": "integer",
                                    "description": "The character count of the content"
                                }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "gpters.automationpro.online",
	BasePath:         "/api/v1",
	Schemes:          []string{"https"},
	Title:            "BetterMode API Scraper",
	Description:      "A server that retrieves content from BetterMode API",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  doc,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
