package service

import (
	"errors"

	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

func translatePortfolioError(err error) *helper.AppError {
	switch {
	case errors.Is(err, repository.ErrPortfolioNotFound):
		return helper.NotFound("portofolio tidak ditemukan")
	case errors.Is(err, repository.ErrMaxProjectsReached):
		return helper.Conflict("portofolio sudah mencapai batas 20 proyek")
	case errors.Is(err, repository.ErrProjectNotFound):
		return helper.NotFound("proyek tidak ditemukan")
	default:
		return helper.Internal(err)
	}
}
