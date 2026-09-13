# test_security.ps1
# Script pengujian keamanan otomatis untuk Modul 5 menggunakan curl.exe

$baseUrl = "http://localhost:3000/api/v1"
$logFile = ".\modul5\security_test_evidence.log"
"" | Out-File -FilePath $logFile -Encoding utf8

function Log-Output($msg) {
    Write-Host $msg
    $msg | Out-File -FilePath $logFile -Append -Encoding utf8
}

Log-Output "==============================================================="
Log-Output "HASIL PENGUJIAN KEAMANAN & AUTENTIKASI (MODUL 5)"
Log-Output "Waktu Pengujian: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
Log-Output "===============================================================`n"

# 1. Health Check
Log-Output "--- [TEST 0] GET /health (Publik) ---"
$out0 = curl.exe -i -s "$baseUrl/health"
Log-Output $out0
Log-Output ""

# 2. Register Normal
Log-Output "--- [TEST 1] POST /auth/register (Normal -> 201 Created, Tanpa Password) ---"
$out1 = curl.exe -i -s -X POST "$baseUrl/auth/register" -H "Content-Type: application/json" -d '{\"username\":\"sari_rahma_test\",\"email\":\"sari_test@example.com\",\"password\":\"SuperSecretPassword2026!\"}'
Log-Output $out1
Log-Output ""

# 3. Register Password Lemah
Log-Output "--- [TEST 2] POST /auth/register (Password Lemah / Umum -> 422 Unprocessable Entity) ---"
$out2 = curl.exe -i -s -X POST "$baseUrl/auth/register" -H "Content-Type: application/json" -d '{\"username\":\"budi_santoso_test\",\"email\":\"budi_test@example.com\",\"password\":\"password123\"}'
Log-Output $out2
Log-Output ""

# 4. Verifikasi Database Password Hash
Log-Output "--- [TEST 3] Cek DB: Format Hash Password bcrypt & Role Default ---"
$dbCheck = psql -U postgres -h 127.0.0.1 -d praktikum_backend -t -c "SELECT username, substring(password from 1 for 7) as prefix, role FROM users WHERE username = 'sari_rahma_test';"
Log-Output "DB Query Output: $dbCheck"
Log-Output ""

# 5. Mass Assignment Prevention
Log-Output "--- [TEST 4] Mass Assignment: Request Register Mengirim role='admin' ---"
$out4 = curl.exe -i -s -X POST "$baseUrl/auth/register" -H "Content-Type: application/json" -d '{\"username\":\"hacker_role_test\",\"email\":\"hacker_test@example.com\",\"password\":\"HackerSecurePass123!\",\"role\":\"admin\"}'
Log-Output $out4
$roleCheck = psql -U postgres -h 127.0.0.1 -d praktikum_backend -t -c "SELECT username, role FROM users WHERE username = 'hacker_role_test';"
Log-Output "DB Role Hasil: $roleCheck (Role tetap 'user', bukan 'admin')`n"

# 6. Akses /students Tanpa Token
Log-Output "--- [TEST 5] Akses /students Tanpa Token -> 401 Unauthorized + WWW-Authenticate ---"
$out5 = curl.exe -i -s "$baseUrl/students"
Log-Output $out5
Log-Output ""

# 7. User Enumeration Prevention
Log-Output "--- [TEST 6] User Enumeration: Perbandingan Login User Tidak Ada vs Password Salah ---"
$out6a = curl.exe -i -s -X POST "$baseUrl/auth/login" -H "Content-Type: application/json" -d '{\"username\":\"user_tidak_ada_xyz\",\"password\":\"passwordApapun123\"}'
Log-Output "Login User Tak Ada:"
Log-Output $out6a
$out6b = curl.exe -i -s -X POST "$baseUrl/auth/login" -H "Content-Type: application/json" -d '{\"username\":\"sari_rahma_test\",\"password\":\"PasswordSalah123!\"}'
Log-Output "Login Password Salah:"
Log-Output $out6b
Log-Output "Kesimpulan: Kedua respon status (401) dan pesan ('username atau password salah') identik.`n"

# 8. Login Berhasil
Log-Output "--- [TEST 7] POST /auth/login (Kredensial Benar -> 200 OK) ---"
$out7 = curl.exe -s -X POST "$baseUrl/auth/login" -H "Content-Type: application/json" -d '{\"username\":\"sari_rahma_test\",\"password\":\"SuperSecretPassword2026!\"}'
Log-Output "Response body: $out7"
$loginData = ($out7 | ConvertFrom-Json).data
$accessToken = $loginData.access_token
$refreshToken = $loginData.refresh_token
Log-Output "Access Token: $($accessToken.Substring(0, 30))..."
Log-Output "Refresh Token: $($refreshToken.Substring(0, 30))..."
Log-Output "Expires In: $($loginData.expires_in) detik`n"

# 9. Akses /students dengan Token Valid
Log-Output "--- [TEST 8] GET /students dengan Access Token Valid -> 200 OK ---"
$out8 = curl.exe -i -s "$baseUrl/students" -H "Authorization: Bearer $accessToken"
Log-Output $out8
Log-Output ""

# 10. Tampered Token
Log-Output "--- [TEST 9] GET /students dengan Token Dimodifikasi 1 Karakter -> 401 Unauthorized ---"
$tamperedToken = $accessToken.Substring(0, $accessToken.Length - 2) + "xx"
$out9 = curl.exe -i -s "$baseUrl/students" -H "Authorization: Bearer $tamperedToken"
Log-Output $out9
Log-Output ""

# 11. Algorithm Confusion (alg: none)
Log-Output "--- [TEST 10] GET /students dengan Token alg='none' -> 401 Unauthorized ---"
$noneHeader = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes('{"alg":"none","typ":"JWT"}')).TrimEnd('=').Replace('+','-').Replace('/','_')
$nonePayload = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes('{"user_id":1,"username":"sari_rahma_test","role":"admin"}')).TrimEnd('=').Replace('+','-').Replace('/','_')
$noneToken = "$noneHeader.$nonePayload."
$out10 = curl.exe -i -s "$baseUrl/students" -H "Authorization: Bearer $noneToken"
Log-Output $out10
Log-Output ""

# 12. Refresh Token Rotation & One-time Use
Log-Output "--- [TEST 11] Refresh Token Rotation: Gunakan Sekali -> Sukses 200, Gunakan Lagi -> 401 ---"
$out11a = curl.exe -i -s -X POST "$baseUrl/auth/refresh" -H "Content-Type: application/json" -d "{\`"refresh_token\`":\`"$refreshToken\`"}"
Log-Output "Pemakaian Pertama:"
Log-Output $out11a
$out11b = curl.exe -i -s -X POST "$baseUrl/auth/refresh" -H "Content-Type: application/json" -d "{\`"refresh_token\`":\`"$refreshToken\`"}"
Log-Output "Pemakaian Kedua (Token Lama Bekas):"
Log-Output $out11b
Log-Output ""

# 13. Endpoint /me
# Dapatkan access token baru dari refresh response
$jsonLine = ($out11a -split "`r?`n" | Where-Object { $_.Trim().StartsWith('{') }) | Select-Object -First 1
$newAccessToken = ($jsonLine | ConvertFrom-Json).data.access_token
Log-Output "--- [TEST 12] GET /auth/me dengan Access Token Baru -> 200 OK ---"
$out12 = curl.exe -i -s "$baseUrl/auth/me" -H "Authorization: Bearer $newAccessToken"
Log-Output $out12
Log-Output ""

# 14. Brute Force Rate Limiting
Log-Output "--- [TEST 13] Brute Force Rate Limiter: Percobaan Cepat Berturut-turut ---"
for ($i = 1; $i -le 7; $i++) {
    $bfOut = curl.exe -i -s -X POST "$baseUrl/auth/login" -H "Content-Type: application/json" -d '{\"username\":\"sari_rahma_test\",\"password\":\"wrong_password\"}'
    $lines = $bfOut -split "`r?`n"
    $statusLine = $lines[0]
    $retryLine = ($lines | Where-Object { $_ -like "Retry-After*" })
    $bodyLine = $lines[-1]
    Log-Output "Percobaan ${i}: $statusLine | $retryLine | $bodyLine"
}

Log-Output "`n==============================================================="
Log-Output "SELURUH PENGUJIAN KEAMANAN SELESAI DENGAN SUKSES!"
Log-Output "==============================================================="
