package service

import (
	repo "bookstore-backend/internal/domains/promotion/repository"
)

type PromotionService struct {
	repository repo.PromotionRepository
}
