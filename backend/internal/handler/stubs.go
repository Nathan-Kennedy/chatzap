package handler

import "github.com/gofiber/fiber/v2"

// HandleListFlows GET /api/v1/flows
func HandleListFlows() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return JSONSuccess(c, []fiber.Map{})
	}
}

// HandleListCampaigns GET /api/v1/campaigns
func HandleListCampaigns() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return JSONSuccess(c, []fiber.Map{})
	}
}

// HandleKanbanBoard GET /api/v1/kanban/board
func HandleKanbanBoard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return JSONSuccess(c, fiber.Map{
			"columns": []fiber.Map{},
			"cards":   []fiber.Map{},
		})
	}
}
