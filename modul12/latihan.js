// latihan.js - Latihan Operasi Dasar MongoDB untuk Modul 10
// Menjalankan operasi CRUD, update atomik, indeks unik gabungan, multikey, dan ObjectId timestamp.

// Gunakan database praktikum_backend
db = db.getSiblingDB("praktikum_backend_latihan");

print("=== [1] Membersihkan collection ulasan untuk latihan ===");
db.course_reviews.drop();

print("\n=== [2] Membuat Indeks Unik Gabungan & Multikey ===");
// Indeks unik gabungan: satu ulasan per mahasiswa per kelas
db.course_reviews.createIndex(
    { course_id: 1, user_id: 1 },
    { unique: true, name: "uniq_course_user_review" }
);

// Indeks multikey pada array tags
db.course_reviews.createIndex(
    { course_id: 1, tags: 1 },
    { name: "idx_course_tags" }
);

print("\n=== [3] Menyisipkan Dokumen Ulasan (Insert) ===");
const insertResult = db.course_reviews.insertOne({
    course_id: 1,
    user_id: 10,
    username: "user_bambang",
    rating: 5,
    comment: "Mata kuliah Pemrograman Dasar sangat terstruktur dan mudah dipahami.",
    tags: ["dosen_asik", "materi_jelas", "tugas_wajar"],
    helpful_count: 0,
    helpful_users: [],
    replies: [],
    created_at: new Date(),
    updated_at: new Date()
});
print("Dokumen berhasil disisipkan dengan _id: " + insertResult.insertedId);

print("\n=== [4] Membaca Waktu Pembuatan dari ObjectId (Timestamp) ===");
const doc = db.course_reviews.findOne({ _id: insertResult.insertedId });
print("Waktu dari ObjectId: " + doc._id.getTimestamp().toISOString());

print("\n=== [5] Menambahkan Balasan Bersarang ($push) ===");
db.course_reviews.updateOne(
    { _id: insertResult.insertedId },
    {
        $push: {
            replies: {
                id: new ObjectId(),
                user_id: 7,
                username: "admin_budi",
                comment: "Terima kasih atas ulasan positifnya!",
                created_at: new Date()
            }
        }
    }
);

print("\n=== [6] Menandai Membantu Secara Atomik ($addToSet & $inc) ===");
// Tandai membantu oleh user 9 (andi)
db.course_reviews.updateOne(
    { _id: insertResult.insertedId, helpful_users: { $ne: 9 } },
    {
        $addToSet: { helpful_users: 9 },
        $inc: { helpful_count: 1 }
    }
);

print("\n=== [7] Pembuktian Indeks Unik Gabungan (Tolak Ulasan Ganda) ===");
try {
    db.course_reviews.insertOne({
        course_id: 1,
        user_id: 10, // user_id sama di course_id yang sama -> HARUS DITOLAK
        username: "user_bambang",
        rating: 4,
        comment: "Ulasan kedua dari user yang sama seharusnya ditolak",
        tags: ["duplikat"],
        created_at: new Date()
    });
    print("GAGAL: Duplikasi diterima!");
} catch (e) {
    print("SUKSES: Duplikasi berhasil ditolak oleh indeks unik! Kode error: " + e.code);
}

print("\n=== [8] Eksekusi explain('executionStats') Membuktikan IXSCAN ===");
const explainStats = db.course_reviews.find({ course_id: 1, user_id: 10 }).explain("executionStats");
print("Tahap Eksekusi (Stage): " + explainStats.executionStats.executionStages.stage);
print("Total Dokumen Diperiksa: " + explainStats.executionStats.totalDocsExamined);
print("Total Dokumen Dikembalikan: " + explainStats.executionStats.nReturned);

print("\n=== Latihan mongosh selesai dengan sukses! ===");
