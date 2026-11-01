package helper

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"modul10/app/model"
)

func ParamID(c *fiber.Ctx) (int, bool) {
	raw := c.Params("id")
	if raw == "" {
		return 0, false
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   strings.ToLower(strings.TrimSpace(c.Query("sort", "id"))),
		Order:  strings.ToLower(strings.TrimSpace(c.Query("order", "asc"))),
	}

	if q.Page < 1 {
		q.Page = 1
	}

	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	if q.Order != "desc" {
		q.Order = "asc"
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}

	if raw := c.Query("grade_min"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.GradeMin = &v
		}
	}
	if raw := c.Query("grade_max"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.GradeMax = &v
		}
	}

	return q
}

func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	q := model.CursorQuery{
		Limit:  c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
	}

	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return q, BadRequest("is_active harus berupa true atau false")
		}
		q.IsActive = &v
	}

	cursorStr := c.Query("cursor")
	if cursorStr == "" {
		cursorStr = c.Query("after")
	}

	if cursorStr != "" {
		cursor, err := DecodeCursor(cursorStr)
		if err != nil {
			if errors.Is(err, ErrInvalidCursor) {
				return q, BadRequest("cursor tidak valid")
			}
			return q, BadRequest("cursor tidak valid")
		}
		q.After = &cursor
	}

	return q, nil
}
