package helper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"latihan-fiber/app/model"
)

// Dua golongan kegagalan pengiriman. Pembedaan ini menentukan apakah
// sebuah task layak di-retry oleh queue broker (Asynq).
var (
	// ErrTemporaryDelivery: server email sedang sibuk atau jaringan putus.
	// Melakukan retry beberapa saat kemudian kemungkinan besar berhasil.
	ErrTemporaryDelivery = errors.New("pengiriman gagal sementara")

	// ErrPermanentDelivery: alamat tujuan tidak ada atau format tidak valid.
	// Melakukan retry berkali-kali pun hasilnya akan tetap sama.
	ErrPermanentDelivery = errors.New("pengiriman gagal permanen")
)

// Mailer adalah interface kontrak pengiriman pesan email.
// Service hanya bergantung pada antarmuka ini, sehingga implementasi LogMailer
// tiruan dapat dengan mudah diganti dengan mailer SMTP asli tanpa mengubah logika service.
type Mailer interface {
	Send(ctx context.Context, msg model.Message) error
}

// LogMailer adalah mailer tiruan untuk lingkungan praktikum.
// LogMailer tidak mengirim email nyata ke internet, melainkan menuliskan log transaksi ke logs/mail.log,
// sekaligus mensimulasikan latensi pengiriman serta tingkat kegagalan acak/periodik.
type LogMailer struct {
	mu        sync.Mutex
	path      string
	delay     time.Duration
	failEvery int64
	attempts  atomic.Int64
}

// NewLogMailer membuat instance LogMailer baru.
//
//   - path      : jalur berkas penyimpanan log email (misal: logs/mail.log).
//   - delay     : lama durasi simulasi satu kali pengiriman email (latensi).
//   - failEvery : simulasi kegagalan berkala; setiap percobaan ke-N gagal sementara (0 = tidak pernah gagal).
//
// Catatan Perbaikan Puzzle 1: Mengembalikan pointer (&LogMailer), bukan struct value.
// Karena method Send memiliki pointer receiver (*LogMailer) dan membawa sync.Mutex & atomic.Int64
// yang tidak boleh disalin (copied by value) oleh goroutine worker yang berjalan secara konkruen.
func NewLogMailer(path string, delay time.Duration, failEvery int) (Mailer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("menyiapkan folder log email: %w", err)
	}

	// Mengembalikan pointer ke instance LogMailer agar memenuhi antarmuka Mailer
	return &LogMailer{
		path:      path,
		delay:     delay,
		failEvery: int64(failEvery),
	}, nil
}

// Send mensimulasikan pengiriman satu pesan email.
// Memeriksa pembatalan context, validasi domain .invalid (RFC 2606), dan simulasi kegagalan berkala.
func (m *LogMailer) Send(ctx context.Context, msg model.Message) error {
	n := m.attempts.Add(1)

	// Meniru latensi jaringan ke server email.
	// Bila context dibatalkan (misalnya timeout task atau sinyal shutdown), pengiriman segera dibatalkan.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(m.delay):
	}

	// Domain .invalid dicadangkan oleh RFC 2606 sebagai domain yang dijamin tidak pernah ada.
	// Digunakan untuk menguji skenario permanent delivery failure.
	if strings.HasSuffix(strings.ToLower(msg.To), ".invalid") {
		return fmt.Errorf("%w: alamat %s tidak ditemukan", ErrPermanentDelivery, msg.To)
	}

	// Simulasi temporary failure: jika failEvery > 0 dan kelipatan attempt ke-N tercapai
	if m.failEvery > 0 && n%m.failEvery == 0 {
		return fmt.Errorf("%w: server email sibuk (attempt ke-%d)", ErrTemporaryDelivery, n)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	f, err := os.OpenFile(m.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTemporaryDelivery, err)
	}
	defer f.Close()

	// Hanya mencatat waktu, penerima (to), dan subjek email demi privasi (tidak mencatat body mentah).
	_, err = fmt.Fprintf(f, "%s to=%s subject=%q\n",
		time.Now().Format(time.RFC3339), msg.To, msg.Subject)
	return err
}

