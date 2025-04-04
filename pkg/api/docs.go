package api

import "github.com/swaggo/swag"

// @title QNQA Auto Crawlers API
// @version 1.0
// @description API для управления краулерами автомобильных сайтов
// @host localhost:8080
// @BasePath /
// @schemes http https
func SwaggerInfo() {
	swag.Register(swag.Name, &swag.Spec{
		InfoInstanceName: swag.Name,
		SwaggerTemplate:  docTemplate,
	})
}

const docTemplate = `{
    "swagger": "2.0",
    "info": {
        "title": "QNQA Auto Crawlers API",
        "description": "API для управления краулерами автомобильных сайтов",
        "version": "1.0"
    },
    "host": "localhost:8080",
    "basePath": "/",
    "schemes": [
        "http",
        "https"
    ],
    "paths": {
        "/api/mobilede/parse-brands": {
            "post": {
                "summary": "Парсинг брендов",
                "description": "Запускает процесс парсинга брендов с MobileDE",
                "tags": [
                    "MobileDE"
                ],
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "responses": {
                    "200": {
                        "description": "Успешный ответ",
                        "schema": {
                            "$ref": "#/definitions/Response"
                        }
                    }
                }
            }
        },
        "/api/mobilede/parse-models": {
            "post": {
                "summary": "Парсинг моделей",
                "description": "Запускает процесс парсинга моделей с MobileDE",
                "tags": [
                    "MobileDE"
                ],
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "responses": {
                    "200": {
                        "description": "Успешный ответ",
                        "schema": {
                            "$ref": "#/definitions/Response"
                        }
                    },
                    "400": {
                        "description": "Ошибка запроса",
                        "schema": {
                            "$ref": "#/definitions/Response"
                        }
                    }
                }
            }
        },
        "/api/mobilede/check-partitions": {
            "post": {
                "summary": "Проверка и создание партиций",
                "description": "Проверяет наличие партиций для брендов и создает их, если они отсутствуют",
                "tags": [
                    "MobileDE"
                ],
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "responses": {
                    "200": {
                        "description": "Успешный ответ",
                        "schema": {
                            "$ref": "#/definitions/Response"
                        }
                    },
                    "400": {
                        "description": "Ошибка запроса",
                        "schema": {
                            "$ref": "#/definitions/Response"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "Response": {
            "type": "object",
            "properties": {
                "success": {
                    "type": "boolean",
                    "description": "Успешность операции"
                },
                "message": {
                    "type": "string",
                    "description": "Сообщение о результате операции"
                },
                "data": {
                    "type": "object",
                    "description": "Данные ответа"
                }
            }
        }
    }
}`
