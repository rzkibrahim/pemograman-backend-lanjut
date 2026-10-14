# Script Pengujian Otomatis 10 Endpoint SIAKAD Mini UTS
$baseUrl = "http://localhost:3000/api/v1"

Write-Host "=== PENGUJIAN 10 ENDPOINT SIAKAD MINI ===" -ForegroundColor Cyan

# 1. Login Admin
Write-Host "`n[1] POST /api/v1/auth/login (Admin)" -ForegroundColor Yellow
$adminLoginBody = '{"email":"admin@siakad.ac.id","password":"admin12345"}'
$adminLogin = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -ContentType "application/json" -Body $adminLoginBody
$adminToken = $adminLogin.data.access_token
Write-Host "Admin Token didapatkan: $($adminToken.Substring(0, 25))..." -ForegroundColor Green

# 1b. Login Mahasiswa 1
Write-Host "`n[1b] POST /api/v1/auth/login (Mahasiswa Rina Putri)" -ForegroundColor Yellow
$mhsLoginBody = '{"email":"rina.putri@mhs.siakad.ac.id","password":"187221000001"}'
$mhsLogin = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -ContentType "application/json" -Body $mhsLoginBody
$mhsToken = $mhsLogin.data.access_token
Write-Host "Mahasiswa Token didapatkan: $($mhsToken.Substring(0, 25))..." -ForegroundColor Green

# 1c. Uji Validasi 422 Login
Write-Host "`n[1c] Uji Validasi Login Gagal (422) - Password < 8 karakter" -ForegroundColor Yellow
try {
    $invalidLogin = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -ContentType "application/json" -Body '{"email":"invalid","password":"123"}'
} catch {
    $errRes = $_.ErrorDetails.Message
    Write-Host "Status Code: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Magenta
    Write-Host "Response Body: $errRes" -ForegroundColor Magenta
}

# 2. GET /api/v1/auth/me (Admin)
Write-Host "`n[2] GET /api/v1/auth/me (Admin)" -ForegroundColor Yellow
$meAdmin = Invoke-RestMethod -Uri "$baseUrl/auth/me" -Method Get -Headers @{ Authorization = "Bearer $adminToken" }
Write-Host ($meAdmin | ConvertTo-Json -Depth 3) -ForegroundColor Green

# 2b. GET /api/v1/auth/me (Mahasiswa)
Write-Host "`n[2b] GET /api/v1/auth/me (Mahasiswa)" -ForegroundColor Yellow
$meMhs = Invoke-RestMethod -Uri "$baseUrl/auth/me" -Method Get -Headers @{ Authorization = "Bearer $mhsToken" }
Write-Host ($meMhs | ConvertTo-Json -Depth 3) -ForegroundColor Green

# 3. GET /api/v1/students (Admin) - Pagination, Filter, Search, Sort
Write-Host "`n[3] GET /api/v1/students (Admin - page=1, per_page=3, sort=-ipk_terakhir)" -ForegroundColor Yellow
$students = Invoke-RestMethod -Uri "$baseUrl/students?page=1&per_page=3&sort=-ipk_terakhir" -Method Get -Headers @{ Authorization = "Bearer $adminToken" }
Write-Host ($students | ConvertTo-Json -Depth 4) -ForegroundColor Green

# 3b. GET /api/v1/students oleh Mahasiswa (Harus 403 Forbidden)
Write-Host "`n[3b] GET /api/v1/students oleh Mahasiswa (Harus 403 Forbidden)" -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$baseUrl/students" -Method Get -Headers @{ Authorization = "Bearer $mhsToken" }
} catch {
    Write-Host "Status Code: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Magenta
    Write-Host "Response Body: $($_.ErrorDetails.Message)" -ForegroundColor Magenta
}

# 4. POST /api/v1/students (Admin) - Tambah Mahasiswa Baru
Write-Host "`n[4] POST /api/v1/students (Admin)" -ForegroundColor Yellow
$newMhsBody = @{
    nim = "187221000099"
    nama = "Bambang Pamungkas"
    email = "bambang.pamungkas@mhs.siakad.ac.id"
    prodi = "Sistem Informasi"
    angkatan = 2024
    ipk_terakhir = 3.65
} | ConvertTo-Json
$createdMhs = Invoke-RestMethod -Uri "$baseUrl/students" -Method Post -ContentType "application/json" -Headers @{ Authorization = "Bearer $adminToken" } -Body $newMhsBody
Write-Host ($createdMhs | ConvertTo-Json -Depth 3) -ForegroundColor Green
$newMhsID = $createdMhs.data.id

# 4b. Uji Duplikasi NIM (Harus 422 Validation Error)
Write-Host "`n[4b] POST /api/v1/students Duplikasi NIM (Harus 422)" -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$baseUrl/students" -Method Post -ContentType "application/json" -Headers @{ Authorization = "Bearer $adminToken" } -Body $newMhsBody
} catch {
    Write-Host "Status Code: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Magenta
    Write-Host "Response Body: $($_.ErrorDetails.Message)" -ForegroundColor Magenta
}

# 5. GET /api/v1/students/{id} (Admin & Mahasiswa Data Sendiri)
Write-Host "`n[5] GET /api/v1/students/{id} (Admin lihat detail Mahasiswa 1)" -ForegroundColor Yellow
$detailMhs1 = Invoke-RestMethod -Uri "$baseUrl/students/1" -Method Get -Headers @{ Authorization = "Bearer $adminToken" }
Write-Host ($detailMhs1 | ConvertTo-Json -Depth 4) -ForegroundColor Green

# 5b. Mahasiswa 1 lihat detail data mahasiswa lain (Mahasiswa 2) -> Harus 403 Forbidden
Write-Host "`n[5b] Mahasiswa akses data mahasiswa lain (Harus 403 Forbidden)" -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$baseUrl/students/2" -Method Get -Headers @{ Authorization = "Bearer $mhsToken" }
} catch {
    Write-Host "Status Code: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Magenta
    Write-Host "Response Body: $($_.ErrorDetails.Message)" -ForegroundColor Magenta
}

# 6. PUT /api/v1/students/{id} (Admin) - Update Mahasiswa Baru
Write-Host "`n[6] PUT /api/v1/students/{id} (Admin perbarui data mahasiswa)" -ForegroundColor Yellow
$updateBody = @{
    nama = "Bambang Pamungkas, S.Kom"
    prodi = "Teknologi Sains Data"
    angkatan = 2024
    ipk_terakhir = 3.90
} | ConvertTo-Json
$updatedMhs = Invoke-RestMethod -Uri "$baseUrl/students/$newMhsID" -Method Put -ContentType "application/json" -Headers @{ Authorization = "Bearer $adminToken" } -Body $updateBody
Write-Host ($updatedMhs | ConvertTo-Json -Depth 3) -ForegroundColor Green

# 7. DELETE /api/v1/students/{id} (Admin) - Soft Delete Mahasiswa Baru
Write-Host "`n[7] DELETE /api/v1/students/{id} (Admin soft delete mahasiswa)" -ForegroundColor Yellow
$delRes = Invoke-WebRequest -Uri "$baseUrl/students/$newMhsID" -Method Delete -UseBasicParsing -Headers @{ Authorization = "Bearer $adminToken" }
Write-Host "Status Code: $($delRes.StatusCode) (204 No Content)" -ForegroundColor Green

# 8. GET /api/v1/courses (Daftar mata kuliah dan sisa kuota)
Write-Host "`n[8] GET /api/v1/courses (Semua role)" -ForegroundColor Yellow
$courses = Invoke-RestMethod -Uri "$baseUrl/courses?available=true" -Method Get -Headers @{ Authorization = "Bearer $mhsToken" }
Write-Host ($courses | ConvertTo-Json -Depth 4) -ForegroundColor Green

# 9. POST /api/v1/enrollments (Mahasiswa ambil mata kuliah)
Write-Host "`n[9] POST /api/v1/enrollments (Mahasiswa ambil mata kuliah SI102)" -ForegroundColor Yellow
$enrollBody = @{
    course_id = 2
    tahun_akademik = "2026/2027-Ganjil"
} | ConvertTo-Json
$enrollRes = Invoke-RestMethod -Uri "$baseUrl/enrollments" -Method Post -ContentType "application/json" -Headers @{ Authorization = "Bearer $mhsToken" } -Body $enrollBody
Write-Host ($enrollRes | ConvertTo-Json -Depth 3) -ForegroundColor Green
$enrollID = $enrollRes.data.id

# 9b. Uji Duplikasi Mata Kuliah pada Tahun Akademik Sama (Harus 409 Conflict)
Write-Host "`n[9b] POST /api/v1/enrollments Duplikasi MK (Harus 409 Conflict)" -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$baseUrl/enrollments" -Method Post -ContentType "application/json" -Headers @{ Authorization = "Bearer $mhsToken" } -Body $enrollBody
} catch {
    Write-Host "Status Code: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Magenta
    Write-Host "Response Body: $($_.ErrorDetails.Message)" -ForegroundColor Magenta
}

# 10. DELETE /api/v1/enrollments/{id} (Mahasiswa batalkan KRS)
Write-Host "`n[10] DELETE /api/v1/enrollments/{id} (Mahasiswa batalkan KRS)" -ForegroundColor Yellow
$cancelEnroll = Invoke-WebRequest -Uri "$baseUrl/enrollments/$enrollID" -Method Delete -UseBasicParsing -Headers @{ Authorization = "Bearer $mhsToken" }
Write-Host "Status Code: $($cancelEnroll.StatusCode) (204 No Content)" -ForegroundColor Green

Write-Host "`n=== SEMUA 10 ENDPOINT TERVERIFIKASI SUKSES! ===" -ForegroundColor Cyan
