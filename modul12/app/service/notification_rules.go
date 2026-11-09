package service

import (
	"errors"
	"fmt"

	"latihan-fiber/app/model"
)

var errUnknownEvent = errors.New("jenis event tidak dikenal")

// BuildMessage menyusun pesan notifikasi/email dari sebuah EnrollmentEvent.
// Fungsi ini murni (pure function): tidak menyentuh database, Redis, maupun sistem IO,
// sehingga sangat mudah diuji secara terisolasi.
func BuildMessage(eventType string, e model.EnrollmentEvent, user model.User) (model.Message, error) {
	course := e.CourseCode + " " + e.CourseName

	var subject, body string
	switch eventType {
	case model.EventEnrollmentCreated:
		subject = "KRS: berhasil terdaftar di " + course
		body = fmt.Sprintf("Halo %s, Anda tercatat sebagai peserta kelas %s.", user.Username, course)
	case model.EventEnrollmentDropped:
		subject = "KRS: keluar dari " + course
		body = fmt.Sprintf("Halo %s, Anda sudah tidak terdaftar di kelas %s.", user.Username, course)
	case model.EventEnrollmentSwapped:
		from := e.FromCourseCode + " " + e.FromCourseName
		subject = "KRS: pindah dari " + from + " ke " + course
		body = fmt.Sprintf("Halo %s, Anda telah pindah dari kelas %s ke kelas %s.", user.Username, from, course)
	default:
		return model.Message{}, fmt.Errorf("%w: %q", errUnknownEvent, eventType)
	}
	return model.Message{To: user.Email, Subject: subject, Body: body}, nil
}

// BuildQuotaChangedMessage menyusun pesan notifikasi perubahan kuota untuk Tugas Mandiri D.2.
// Format subjek sesuai modul: "KRS: kuota <kode> <nama> berubah menjadi <kuota>"
func BuildQuotaChangedMessage(e model.CourseQuotaChangedEvent, user model.User) model.Message {
	subject := fmt.Sprintf("KRS: kuota %s %s berubah menjadi %d", e.CourseCode, e.CourseName, e.NewQuota)
	body := fmt.Sprintf("Halo %s, kuota kelas %s %s telah diperbarui oleh akademik menjadi %d peserta.",
		user.Username, e.CourseCode, e.CourseName, e.NewQuota)
	return model.Message{To: user.Email, Subject: subject, Body: body}
}

