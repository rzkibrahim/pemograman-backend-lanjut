$baseUrl = "http://localhost:3000/api/v1"

Write-Host "=== 1. Health Check ==="
$h = Invoke-RestMethod -Uri "$baseUrl/health" -Method Get
Write-Host "Health:" ($h | ConvertTo-Json -Compress)

Write-Host "`n=== 2. Login Admin ==="
$loginBody = @{
    username = "admin_budi"
    password = "Password123!"
} | ConvertTo-Json
$loginRes = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -Body $loginBody -ContentType "application/json"
$token = $loginRes.data.access_token
Write-Host "Admin Login Success. Token length:" $token.Length

$headers = @{
    Authorization = "Bearer $token"
}

Write-Host "`n=== 3. Content Negotiation GET /students ==="
Write-Host "--- 3.1 Accept: application/json ---"
$rJson = Invoke-WebRequest -Uri "$baseUrl/students?limit=2" -Method Get -Headers $headers -ContentType "application/json"
Write-Host "Status:" $rJson.StatusCode
Write-Host "Content-Type:" $rJson.Headers["Content-Type"]
Write-Host "Body snippet:" ($rJson.Content.Substring(0, [Math]::Min(150, $rJson.Content.Length)))

Write-Host "`n--- 3.2 Accept: text/csv ---"
$headersCsv = @{
    Authorization = "Bearer $token"
    Accept = "text/csv"
}
$rCsv = Invoke-WebRequest -Uri "$baseUrl/students?limit=3" -Method Get -Headers $headersCsv
Write-Host "Status:" $rCsv.StatusCode
Write-Host "Content-Type:" $rCsv.Headers["Content-Type"]
Write-Host "CSV Content:`n" $rCsv.Content

Write-Host "--- 3.3 Accept: */* ---"
$headersWildcard = @{
    Authorization = "Bearer $token"
    Accept = "*/*"
}
$rWildcard = Invoke-WebRequest -Uri "$baseUrl/students?limit=2" -Method Get -Headers $headersWildcard
Write-Host "Status:" $rWildcard.StatusCode
Write-Host "Content-Type:" $rWildcard.Headers["Content-Type"]

Write-Host "`n--- 3.4 Accept: application/xml (Expect 406 Not Acceptable) ---"
$headersXml = @{
    Authorization = "Bearer $token"
    Accept = "application/xml"
}
try {
    Invoke-WebRequest -Uri "$baseUrl/students?limit=2" -Method Get -Headers $headersXml
} catch {
    Write-Host "Status:" $_.Exception.Response.StatusCode.value__
    $stream = $_.Exception.Response.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($stream)
    Write-Host "Body:" $reader.ReadToEnd()
}

Write-Host "`n=== 4. Cursor Pagination GET /students ==="
Write-Host "--- 4.1 Page 1 (limit=2) ---"
$p1 = Invoke-RestMethod -Uri "$baseUrl/students?limit=2" -Method Get -Headers $headers
Write-Host "Page 1 count:" $p1.data.Count
Write-Host "Items in P1:" ($p1.data | ForEach-Object { "$($_.id): $($_.name)" })
Write-Host "Next Cursor:" $p1.meta.next_cursor

$cursor = $p1.meta.next_cursor

Write-Host "`n--- 4.2 Sisipkan data baru di posisi teratas ---"
$newStudent = @{
    nim = "082311633999"
    name = "Mahasiswa Terbaru Paling Atas"
    grade = 99.0
    is_active = $true
} | ConvertTo-Json
$insertRes = Invoke-RestMethod -Uri "$baseUrl/students" -Method Post -Headers $headers -Body $newStudent -ContentType "application/json"
$newId = $insertRes.data.id
Write-Host "Inserted student ID:" $newId "Name:" $insertRes.data.name

Write-Host "`n--- 4.3 Page 2 dengan Cursor dari Page 1 (limit=2) ---"
$p2 = Invoke-RestMethod -Uri "$baseUrl/students?limit=2&after=$cursor" -Method Get -Headers $headers
Write-Host "Page 2 count:" $p2.data.Count
Write-Host "Items in P2:" ($p2.data | ForEach-Object { "$($_.id): $($_.name)" })

# Check for duplicates between P1 and P2
$p1Ids = $p1.data | ForEach-Object { $_.id }
$p2Ids = $p2.data | ForEach-Object { $_.id }
$overlap = $p1Ids | Where-Object { $p2Ids -contains $_ }
Write-Host "Overlap count between P1 and P2 (Harus 0):" $overlap.Count

Write-Host "`n=== 5. PATCH Student - Pointer & omitnil ==="
Write-Host "--- 5.1 PATCH name string kosong (Harus Ditolak 422) ---"
$patchInvalid = @{
    name = ""
} | ConvertTo-Json
try {
    Invoke-WebRequest -Uri "$baseUrl/students/$newId" -Method Patch -Headers $headers -Body $patchInvalid -ContentType "application/json"
} catch {
    Write-Host "Status:" $_.Exception.Response.StatusCode.value__
    $stream = $_.Exception.Response.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($stream)
    Write-Host "Body:" $reader.ReadToEnd()
}

Write-Host "`n--- 5.2 PATCH tanpa name (Hanya ubah grade - Harus Sukses 200) ---"
$patchValid = @{
    grade = 98.75
} | ConvertTo-Json
$patchRes = Invoke-RestMethod -Uri "$baseUrl/students/$newId" -Method Patch -Headers $headers -Body $patchValid -ContentType "application/json"
Write-Host "Status: 200 OK"
Write-Host "Patched Student Name:" $patchRes.data.name "Grade:" $patchRes.data.grade

Write-Host "`n=== 6. Validasi NIM Format ==="
$nimInvalid = @{
    nim = "123#xyz!"
    name = "Test NIM Invalid"
    grade = 80
} | ConvertTo-Json
try {
    Invoke-WebRequest -Uri "$baseUrl/students" -Method Post -Headers $headers -Body $nimInvalid -ContentType "application/json"
} catch {
    Write-Host "Status:" $_.Exception.Response.StatusCode.value__
    $stream = $_.Exception.Response.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($stream)
    Write-Host "Body:" $reader.ReadToEnd()
}

Write-Host "`n=== 7. Validasi Password Deklaratif ==="
$passCommon = @{
    username = "test_common_pwd"
    email = "testcommon@univ.ac.id"
    password = "password123"
} | ConvertTo-Json
try {
    Invoke-WebRequest -Uri "$baseUrl/auth/register" -Method Post -Body $passCommon -ContentType "application/json"
} catch {
    Write-Host "Status:" $_.Exception.Response.StatusCode.value__
    $stream = $_.Exception.Response.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($stream)
    Write-Host "Body (Common Password):" $reader.ReadToEnd()
}

Write-Host "`n=== 8. Error Seragam 404 & Request ID ==="
try {
    Invoke-WebRequest -Uri "$baseUrl/students/999999" -Method Get -Headers $headers
} catch {
    Write-Host "Status:" $_.Exception.Response.StatusCode.value__
    $stream = $_.Exception.Response.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($stream)
    Write-Host "Body:" $reader.ReadToEnd()
}
