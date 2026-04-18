package handler

import "github.com/gofiber/fiber/v2"

// JSONSuccess envelope alinhado ao playbook: { "data", "meta?" }.
func JSONSuccess(c *fiber.Ctx, data interface{}) error {
	return c.JSON(fiber.Map{"data": data})
}

// JSONError envelope: { "error": { "code", "message", "details?" } }.
func JSONError(c *fiber.Ctx, status int, code, message string, details interface{}) error {
	errObj := fiber.Map{"code": code, "message": message}
	if details != nil {
		errObj["details"] = details
	}
	return c.Status(status).JSON(fiber.Map{"error": errObj})
}
