package server

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"iLean/entity"
	"net/http"
	"strconv"
)

func (s *Server) Controller(c *gin.Context) {

	commandNum, err := strconv.Atoi(c.Param("n"))

	if err != nil {
		c.JSON(http.StatusBadRequest, entity.Response{Status: http.StatusBadRequest, Message: "Status Bad Request"})
		return
	}

	switch commandNum {
	case 1:
		command := new(entity.CommandTemperature)

		c.ShouldBindJSON(&command)

		validate := validator.New()

		err := validate.Struct(command)

		if err != nil {
			c.JSON(http.StatusBadRequest, entity.Response{Status: http.StatusBadRequest, Message: "Status Bad Request"})
			return
		}

		dataBytes, err := json.Marshal(command)

		if err != nil {
			c.JSON(http.StatusInternalServerError, entity.Response{Status: http.StatusInternalServerError, Message: "Status Internal Server Error"})
			return
		}

		s.socket.Send("controller",command.SerialNumber, dataBytes)

		c.JSON(http.StatusOK, entity.Response{Status: http.StatusOK, Message: "ok"})
	case 2:
		command := new(entity.CommandTemperatureBySensor)

		c.ShouldBindJSON(&command)

		validate := validator.New()

		err := validate.Struct(command)

		if err != nil {
			c.JSON(http.StatusBadRequest, entity.Response{Status: http.StatusBadRequest, Message: "Status Bad Request"})
			return
		}

		dataBytes, err := json.Marshal(command)

		if err != nil {
			c.JSON(http.StatusInternalServerError, entity.Response{Status: http.StatusInternalServerError, Message: "Status Internal Server Error"})
			return
		}

		s.socket.Send("controller",command.SerialNumber, dataBytes)

		c.JSON(http.StatusOK, entity.Response{Status: http.StatusOK, Message: "ok"})

	default:
		c.JSON(http.StatusBadRequest, entity.Response{Status: http.StatusBadRequest, Message: "Status Bad Request"})

	}

}
