package scheduler

import (
	"log"
	"time"

	"github.com/Maxito7/hotel_backend/internal/domain"
)

type ReservationScheduler struct {
	reservaRepo domain.ReservaRepository
	ticker      *time.Ticker
}

// NewReservationScheduler crea una nueva instancia del scheduler de reservas
func NewReservationScheduler(reservaRepo domain.ReservaRepository) *ReservationScheduler {
	return &ReservationScheduler{
		reservaRepo: reservaRepo,
	}
}

// Start inicia el scheduler que actualiza reservas expiradas cada 24 horas
func (s *ReservationScheduler) Start() {
	log.Println("🕐 Scheduler de reservas iniciado - Se ejecutará cada 24 horas")

	// Ejecutar inmediatamente al iniciar
	s.UpdateCompletedReservations()

	// Programar ejecución cada 24 horas a las 00:01 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 1, 0, 0, now.Location())
	durationUntilNextRun := time.Until(nextRun)

	log.Printf("⏰ Próxima ejecución programada: %s", nextRun.Format("2006-01-02 15:04:05"))

	// Esperar hasta la próxima ejecución
	time.AfterFunc(durationUntilNextRun, func() {
		s.UpdateCompletedReservations()

		// Luego ejecutar cada 24 horas
		s.ticker = time.NewTicker(24 * time.Hour)
		go func() {
			for range s.ticker.C {
				s.UpdateCompletedReservations()
			}
		}()
	})
}

// Stop detiene el scheduler
func (s *ReservationScheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
		log.Println("🛑 Scheduler de reservas detenido")
	}
}

// UpdateCompletedReservations actualiza las reservas que ya han pasado su fecha de checkout
func (s *ReservationScheduler) UpdateCompletedReservations() {
	log.Println("🔄 Ejecutando actualización de reservas completadas...")

	if err := s.reservaRepo.UpdateExpiredReservations(); err != nil {
		log.Printf("❌ Error actualizando reservas completadas: %v", err)
	} else {
		log.Println("✅ Reservas completadas actualizadas exitosamente")
	}
}
