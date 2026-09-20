# test_rbac.ps1
# Skrip Pengujian Keamanan Komprehensif & Matriks RBAC - Modul 6
# Menguji seluruh kombinasi role (admin, staff, user) x kepemilikan (owner, non-owner) x endpoint
# serta pengujian negatif sesuai checklist C.3.

$ErrorActionPreference = "Continue"
$baseUrl = "http://localhost:3000/api/v1"
$logFile = "$PSScriptRoot\security_test_evidence.log"
"=== BUKTI EKSEKUSI PENGUJIAN MATRIKS RBAC & SECURITY MODUL 6 ===" | Out-File -FilePath $logFile -Encoding utf8
"Tanggal Eksekusi: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" | Out-File -FilePath $logFile -Append -Encoding utf8
"================================================================`n" | Out-File -FilePath $logFile -Append -Encoding utf8

function Log-Test {
    param(
        [string]$TestID,
        [string]$Description,
        [string]$Role,
        [string]$Endpoint,
        [string]$Method,
        [int]$ExpectedStatus,
        [int]$ActualStatus,
        [string]$Details
    )
    $pass = ($ExpectedStatus -eq $ActualStatus)
    $badge = "[FAIL]"
    $color = "Red"
    if ($pass) {
        $badge = "[PASS]"
        $color = "Green"
    }

    $line = "$badge $TestID | Role: $Role | $Method $Endpoint | Expected: $ExpectedStatus | Actual: $ActualStatus | $Description"
    Write-Host $line -ForegroundColor $color
    $line | Out-File -FilePath $logFile -Append -Encoding utf8
    if ($Details) {
        "   Response Detail: $Details" | Out-File -FilePath $logFile -Append -Encoding utf8
    }
}

Write-Host "1. Mempersiapkan akun pengguna uji di basis data..." -ForegroundColor Cyan

# Jalankan seeder Go untuk memastikan password ter-hash bcrypt cost 12 secara valid
go run "$PSScriptRoot\cmd\seed\main.go"

# Ambil ID masing-masing user
$adminID = (psql -U postgres -d praktikum_backend -t -A -c "SELECT id FROM users WHERE username='admin_budi'").Trim()
$staffID = (psql -U postgres -d praktikum_backend -t -A -c "SELECT id FROM users WHERE username='staff_sari'").Trim()
$andiID  = (psql -U postgres -d praktikum_backend -t -A -c "SELECT id FROM users WHERE username='user_andi'").Trim()
$bambangID = (psql -U postgres -d praktikum_backend -t -A -c "SELECT id FROM users WHERE username='user_bambang'").Trim()

Write-Host "User IDs: Admin=$adminID, Staff=$staffID, Andi(User)=$andiID, Bambang(User)=$bambangID" -ForegroundColor Yellow

# Fungsi bantu request API menggunakan System.Net.Http.HttpClient untuk kehandalan penuh
Add-Type -AssemblyName System.Net.Http
$httpClient = New-Object System.Net.Http.HttpClient

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Uri,
        [string]$Token = $null,
        [string]$Body = $null
    )
    
    $req = New-Object System.Net.Http.HttpRequestMessage
    $req.RequestUri = New-Object System.Uri($Uri)
    $req.Method = [System.Net.Http.HttpMethod]::new($Method)

    if ($Token) {
        $req.Headers.Authorization = New-Object System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", $Token)
    }

    if ($Body) {
        $req.Content = New-Object System.Net.Http.StringContent($Body, [System.Text.Encoding]::UTF8, "application/json")
    }

    try {
        $resp = $httpClient.SendAsync($req).GetAwaiter().GetResult()
        $content = $resp.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        return @{
            StatusCode = [int]$resp.StatusCode
            Content    = $content
        }
    } catch {
        return @{
            StatusCode = 0
            Content    = $_.Exception.Message
        }
    }
}

Write-Host "`n2. Mengambil Token JWT untuk masing-masing role..." -ForegroundColor Cyan

# Login Admin
$adminLogin = Invoke-Api -Method POST -Uri "$baseUrl/auth/login" -Body '{"username":"admin_budi","password":"Password123!"}'
$adminObj = $adminLogin.Content | ConvertFrom-Json
$adminToken = $adminObj.data.access_token

# Login Staff
$staffLogin = Invoke-Api -Method POST -Uri "$baseUrl/auth/login" -Body '{"username":"staff_sari","password":"Password123!"}'
$staffObj = $staffLogin.Content | ConvertFrom-Json
$staffToken = $staffObj.data.access_token

# Login User Andi
$andiLogin = Invoke-Api -Method POST -Uri "$baseUrl/auth/login" -Body '{"username":"user_andi","password":"Password123!"}'
$andiObj = $andiLogin.Content | ConvertFrom-Json
$andiToken = $andiObj.data.access_token

# Login User Bambang
$bambangLogin = Invoke-Api -Method POST -Uri "$baseUrl/auth/login" -Body '{"username":"user_bambang","password":"Password123!"}'
$bambObj = $bambangLogin.Content | ConvertFrom-Json
$bambangToken = $bambObj.data.access_token

Write-Host "Token diperoleh: Admin=$([bool]$adminToken), Staff=$([bool]$staffToken), Andi=$([bool]$andiToken), Bambang=$([bool]$bambangToken)" -ForegroundColor Green

Write-Host "`n3. Menjalankan Matriks Pengujian Hak Akses..." -ForegroundColor Cyan

# =========================================================================
# KELOMPOK 1: GET /students (Permission: student:list)
# Diberikan kepada: admin, staff. Ditolak: user, anonymous.
# =========================================================================
Write-Host "`n[A] Pengujian GET /api/v1/students (student:list)" -ForegroundColor Yellow

$res = Invoke-Api -Method GET -Uri "$baseUrl/students" -Token $adminToken
Log-Test -TestID "M1.1" -Description "Admin akses daftar seluruh mahasiswa" -Role "admin" -Endpoint "/students" -Method "GET" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method GET -Uri "$baseUrl/students" -Token $staffToken
Log-Test -TestID "M1.2" -Description "Staff akses daftar seluruh mahasiswa" -Role "staff" -Endpoint "/students" -Method "GET" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method GET -Uri "$baseUrl/students" -Token $andiToken
Log-Test -TestID "M1.3" -Description "User ditolak melihat daftar seluruh mahasiswa" -Role "user" -Endpoint "/students" -Method "GET" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method GET -Uri "$baseUrl/students"
Log-Test -TestID "M1.4" -Description "Tanpa token ditolak autentikasi" -Role "anonymous" -Endpoint "/students" -Method "GET" -ExpectedStatus 401 -ActualStatus $res.StatusCode -Details $res.Content

# =========================================================================
# KELOMPOK 2: POST /students (Permission: student:create)
# Diberikan kepada: admin, staff. Ditolak: user.
# =========================================================================
Write-Host "`n[B] Pengujian POST /api/v1/students (student:create)" -ForegroundColor Yellow

$nimAdmin = "NIM" + (Get-Random -Minimum 10000 -Maximum 99999)
$bodyAdmin = "{`"nim`":`"$nimAdmin`",`"name`":`"Mahasiswa Admin`",`"grade`":90,`"is_active`":true}"
$res = Invoke-Api -Method POST -Uri "$baseUrl/students" -Token $adminToken -Body $bodyAdmin
Log-Test -TestID "M2.1" -Description "Admin berhasil menambah mahasiswa" -Role "admin" -Endpoint "/students" -Method "POST" -ExpectedStatus 201 -ActualStatus $res.StatusCode -Details $res.Content

$nimStaff = "NIM" + (Get-Random -Minimum 10000 -Maximum 99999)
$bodyStaff = "{`"nim`":`"$nimStaff`",`"name`":`"Mahasiswa Staff`",`"grade`":85,`"is_active`":true}"
$res = Invoke-Api -Method POST -Uri "$baseUrl/students" -Token $staffToken -Body $bodyStaff
Log-Test -TestID "M2.2" -Description "Staff berhasil menambah mahasiswa" -Role "staff" -Endpoint "/students" -Method "POST" -ExpectedStatus 201 -ActualStatus $res.StatusCode -Details $res.Content

$nimUser = "NIM" + (Get-Random -Minimum 10000 -Maximum 99999)
$bodyUser = "{`"nim`":`"$nimUser`",`"name`":`"Mahasiswa User`",`"grade`":80,`"is_active`":true}"
$res = Invoke-Api -Method POST -Uri "$baseUrl/students" -Token $andiToken -Body $bodyUser
Log-Test -TestID "M2.3" -Description "User ditolak menambah mahasiswa (tidak punya student:create)" -Role "user" -Endpoint "/students" -Method "POST" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

# =========================================================================
# PENGUJIAN NEGATIF 1: SPOOFING OWNER_ID PADA POST /students (Checklist C.3)
# =========================================================================
Write-Host "`n[C] Pengujian Negatif 1: Spoofing owner_id pada POST /students" -ForegroundColor Yellow

$nimSpoof = "NIM" + (Get-Random -Minimum 10000 -Maximum 99999)
# Staff mencoba menyusupkan "owner_id": 9999 di body
$bodySpoof = "{`"nim`":`"$nimSpoof`",`"name`":`"Mahasiswa Spoof Test`",`"grade`":88,`"is_active`":true,`"owner_id`":9999}"
$res = Invoke-Api -Method POST -Uri "$baseUrl/students" -Token $staffToken -Body $bodySpoof
$createdStudent = ($res.Content | ConvertFrom-Json).data
$spoofSuccess = ($createdStudent.owner_id -eq [int]$staffID)
Log-Test -TestID "NEG-1" -Description "Anti-Spoofing: owner_id dipaksa dari ID token ($staffID), bukan dari body JSON (9999)" -Role "staff" -Endpoint "/students" -Method "POST" -ExpectedStatus 201 -ActualStatus $res.StatusCode -Details "Owner direspons: $($createdStudent.owner_id) (Sesuai ID Token)"

# Siapkan data mahasiswa milik Andi untuk uji kepemilikan (owner_id = andiID)
$nimAndi = "NIM" + (Get-Random -Minimum 10000 -Maximum 99999)
psql -U postgres -d praktikum_backend -c "INSERT INTO students (nim, name, grade, is_active, owner_id) VALUES ('$nimAndi', 'Mahasiswa Milik Andi', 95.0, true, $andiID);"
$studentAndiID = (psql -U postgres -d praktikum_backend -t -A -c "SELECT id FROM students WHERE nim='$nimAndi'").Trim()
Write-Host "Student Andi ID dibuat di DB: $studentAndiID (Owner = $andiID)" -ForegroundColor Yellow

# =========================================================================
# KELOMPOK 3: GET /students/:id (Ownership vs student:read:any)
# =========================================================================
Write-Host "`n[D] Pengujian GET /api/v1/students/:id (Ownership & read:any)" -ForegroundColor Yellow

# Andi melihat datanya sendiri -> Harus Boleh (Owner)
$res = Invoke-Api -Method GET -Uri "$baseUrl/students/$studentAndiID" -Token $andiToken
Log-Test -TestID "M3.1" -Description "Pemilik (user Andi) membaca data mahasiswa miliknya sendiri" -Role "user (owner)" -Endpoint "/students/$studentAndiID" -Method "GET" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

# Bambang (user lain) melihat data Andi -> Harus 403 Forbidden (Bukan pemilik, tidak punya student:read:any)
$res = Invoke-Api -Method GET -Uri "$baseUrl/students/$studentAndiID" -Token $bambangToken
Log-Test -TestID "M3.2" -Description "Bukan pemilik (user Bambang) membaca data milik Andi ditolak" -Role "user (non-owner)" -Endpoint "/students/$studentAndiID" -Method "GET" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

# Staff melihat data Andi -> Harus Boleh (Staff memiliki student:read:any)
$res = Invoke-Api -Method GET -Uri "$baseUrl/students/$studentAndiID" -Token $staffToken
Log-Test -TestID "M3.3" -Description "Staff membaca data mahasiswa milik Andi diizinkan (student:read:any)" -Role "staff (non-owner)" -Endpoint "/students/$studentAndiID" -Method "GET" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

# Admin melihat data Andi -> Harus Boleh (Admin memiliki student:read:any)
$res = Invoke-Api -Method GET -Uri "$baseUrl/students/$studentAndiID" -Token $adminToken
Log-Test -TestID "M3.4" -Description "Admin membaca data mahasiswa milik Andi diizinkan (student:read:any)" -Role "admin (non-owner)" -Endpoint "/students/$studentAndiID" -Method "GET" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

# =========================================================================
# KELOMPOK 4 & PENGUJIAN NEGATIF 2: PUT & PATCH /students/:id
# Ownership vs student:update:any (Checklist C.3)
# =========================================================================
Write-Host "`n[E] Pengujian PUT/PATCH /api/v1/students/:id & Pengujian Negatif 2" -ForegroundColor Yellow

# Andi mengedit datanya sendiri via PUT -> Harus Boleh (Owner)
$putBodyAndi = "{`"nim`":`"$nimAndi`",`"name`":`"Mahasiswa Andi Updated`",`"grade`":98,`"is_active`":true}"
$res = Invoke-Api -Method PUT -Uri "$baseUrl/students/$studentAndiID" -Token $andiToken -Body $putBodyAndi
Log-Test -TestID "M4.1" -Description "Pemilik (user Andi) mengedit datanya sendiri via PUT" -Role "user (owner)" -Endpoint "/students/$studentAndiID" -Method "PUT" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

# PENGUJIAN NEGATIF 2: User lain (Bambang) memanggil PUT pada data Andi
$putBodyBambang = "{`"nim`":`"$nimAndi`",`"name`":`"Hacked by Bambang`",`"grade`":50,`"is_active`":false}"
$res = Invoke-Api -Method PUT -Uri "$baseUrl/students/$studentAndiID" -Token $bambangToken -Body $putBodyBambang
Log-Test -TestID "NEG-2" -Description "Cross-User Mutation: User Bambang mengedit data Andi ditolak (403 Forbidden)" -Role "user (non-owner)" -Endpoint "/students/$studentAndiID" -Method "PUT" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

# Staff memanggil PUT pada data Andi -> Harus 403 Forbidden (Staff TIDAK punya student:update:any!)
$res = Invoke-Api -Method PUT -Uri "$baseUrl/students/$studentAndiID" -Token $staffToken -Body $putBodyBambang
Log-Test -TestID "M4.2" -Description "Staff mengedit data orang lain ditolak (Staff tidak memiliki student:update:any)" -Role "staff (non-owner)" -Endpoint "/students/$studentAndiID" -Method "PUT" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

# Admin memanggil PUT pada data Andi -> Harus Boleh (Admin memiliki student:update:any)
$putBodyAdmin = "{`"nim`":`"$nimAndi`",`"name`":`"Mahasiswa Andi Verified by Admin`",`"grade`":100,`"is_active`":true}"
$res = Invoke-Api -Method PUT -Uri "$baseUrl/students/$studentAndiID" -Token $adminToken -Body $putBodyAdmin
Log-Test -TestID "M4.3" -Description "Admin mengedit data mahasiswa mana pun diizinkan (student:update:any)" -Role "admin (non-owner)" -Endpoint "/students/$studentAndiID" -Method "PUT" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

# =========================================================================
# KELOMPOK 5: DELETE /students/:id (Permission: student:delete)
# Diberikan hanya kepada: admin. Ditolak: staff, user.
# =========================================================================
Write-Host "`n[F] Pengujian DELETE /api/v1/students/:id (student:delete)" -ForegroundColor Yellow

$res = Invoke-Api -Method DELETE -Uri "$baseUrl/students/$studentAndiID" -Token $andiToken
Log-Test -TestID "M5.1" -Description "User mencoba menghapus data mahasiswa ditolak (tidak punya student:delete)" -Role "user" -Endpoint "/students/$studentAndiID" -Method "DELETE" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method DELETE -Uri "$baseUrl/students/$studentAndiID" -Token $staffToken
Log-Test -TestID "M5.2" -Description "Staff mencoba menghapus data mahasiswa ditolak (tidak punya student:delete)" -Role "staff" -Endpoint "/students/$studentAndiID" -Method "DELETE" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method DELETE -Uri "$baseUrl/students/$studentAndiID" -Token $adminToken
Log-Test -TestID "M5.3" -Description "Admin menghapus data mahasiswa diizinkan (memiliki student:delete)" -Role "admin" -Endpoint "/students/$studentAndiID" -Method "DELETE" -ExpectedStatus 204 -ActualStatus $res.StatusCode -Details "No Content (Berhasil)"

# =========================================================================
# KELOMPOK 6: User Management (GET /users, GET/PUT /users/:id, role:assign)
# =========================================================================
Write-Host "`n[G] Pengujian Manajemen User (user:list, user:read:any, role:assign)" -ForegroundColor Yellow

$res = Invoke-Api -Method GET -Uri "$baseUrl/users" -Token $adminToken
Log-Test -TestID "M6.1" -Description "Admin akses daftar seluruh pengguna (user:list)" -Role "admin" -Endpoint "/users" -Method "GET" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method GET -Uri "$baseUrl/users" -Token $staffToken
Log-Test -TestID "M6.2" -Description "Staff akses daftar seluruh pengguna (user:list)" -Role "staff" -Endpoint "/users" -Method "GET" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method GET -Uri "$baseUrl/users" -Token $andiToken
Log-Test -TestID "M6.3" -Description "User ditolak melihat daftar seluruh pengguna" -Role "user" -Endpoint "/users" -Method "GET" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

# Ownership check pada /users/:id
$res = Invoke-Api -Method GET -Uri "$baseUrl/users/$andiID" -Token $andiToken
Log-Test -TestID "M6.4" -Description "User membaca profil miliknya sendiri diizinkan" -Role "user (owner)" -Endpoint "/users/$andiID" -Method "GET" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method GET -Uri "$baseUrl/users/$adminID" -Token $andiToken
Log-Test -TestID "M6.5" -Description "User membaca profil orang lain ditolak" -Role "user (non-owner)" -Endpoint "/users/$adminID" -Method "GET" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

# Role assign: Hanya admin yang punya role:assign
$res = Invoke-Api -Method PATCH -Uri "$baseUrl/users/$bambangID/role" -Token $staffToken -Body '{"role":"staff"}'
Log-Test -TestID "M6.6" -Description "Staff ditolak mengubah role user (tidak punya role:assign)" -Role "staff" -Endpoint "/users/$bambangID/role" -Method "PATCH" -ExpectedStatus 403 -ActualStatus $res.StatusCode -Details $res.Content

$res = Invoke-Api -Method PATCH -Uri "$baseUrl/users/$bambangID/role" -Token $adminToken -Body '{"role":"staff"}'
Log-Test -TestID "M6.7" -Description "Admin berhasil mengubah role user menjadi staff (memiliki role:assign)" -Role "admin" -Endpoint "/users/$bambangID/role" -Method "PATCH" -ExpectedStatus 200 -ActualStatus $res.StatusCode -Details $res.Content

# Kembalikan role Bambang ke user
$res = Invoke-Api -Method PATCH -Uri "$baseUrl/users/$bambangID/role" -Token $adminToken -Body '{"role":"user"}'

Write-Host "`nPengujian selesai! Seluruh bukti hasil eksekusi dicatat ke: $logFile" -ForegroundColor Green
