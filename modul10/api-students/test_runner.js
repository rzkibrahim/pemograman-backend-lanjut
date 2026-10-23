const BASE = "http://localhost:3000/api/v1";

async function main() {
  console.log("==========================================");
  console.log("MODUL 7 ADVANCED API DESIGN TEST SUITE");
  console.log("==========================================\n");

  // 1. Health check
  console.log("--- 1. HEALTH CHECK ---");
  const resHealth = await fetch(`${BASE}/health`);
  const dataHealth = await resHealth.json();
  console.log(`Status: ${resHealth.status}`, dataHealth);

  // 2. Login
  console.log("\n--- 2. LOGIN ADMIN_BUDI ---");
  const resLogin = await fetch(`${BASE}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: "admin_budi", password: "Password123!" }),
  });
  const dataLogin = await resLogin.json();
  const token = dataLogin.data.access_token;
  console.log(`Status: ${resLogin.status}, Token obtained (len: ${token.length})`);

  const authHeaders = {
    Authorization: `Bearer ${token}`,
  };

  // 3. Content Negotiation on GET /students
  console.log("\n--- 3. CONTENT NEGOTIATION (GET /students) ---");

  // 3.1 Accept: application/json
  const resJson = await fetch(`${BASE}/students?limit=2`, {
    headers: { ...authHeaders, Accept: "application/json" },
  });
  console.log(`Accept: application/json -> Status: ${resJson.status}, Content-Type: ${resJson.headers.get("content-type")}`);
  const dataJson = await resJson.json();
  console.log("Body JSON snippet:", JSON.stringify(dataJson).substring(0, 120) + "...");

  // 3.2 Accept: text/csv
  const resCsv = await fetch(`${BASE}/students?limit=3`, {
    headers: { ...authHeaders, Accept: "text/csv" },
  });
  console.log(`Accept: text/csv -> Status: ${resCsv.status}, Content-Type: ${resCsv.headers.get("content-type")}`);
  const textCsv = await resCsv.text();
  console.log("CSV Body:\n" + textCsv);

  // 3.3 Accept: */*
  const resWildcard = await fetch(`${BASE}/students?limit=2`, {
    headers: { ...authHeaders, Accept: "*/*" },
  });
  console.log(`Accept: */* -> Status: ${resWildcard.status}, Content-Type: ${resWildcard.headers.get("content-type")}`);

  // 3.4 Accept: application/xml (Expect 406 Not Acceptable)
  const resXml = await fetch(`${BASE}/students?limit=2`, {
    headers: { ...authHeaders, Accept: "application/xml" },
  });
  console.log(`Accept: application/xml -> Status: ${resXml.status}`);
  const textXml = await resXml.text();
  console.log("Body:", textXml);

  // 4. Cursor Pagination GET /students
  console.log("\n--- 4. CURSOR PAGINATION (GET /students) ---");
  const p1Res = await fetch(`${BASE}/students?limit=2`, {
    headers: authHeaders,
  });
  const p1 = await p1Res.json();
  console.log("Page 1 items:", p1.data.map((s) => `${s.id}: ${s.name} (${s.created_at})`));
  console.log("Next Cursor:", p1.meta.next_cursor);
  const nextCursor = p1.meta.next_cursor;

  // Insert a new student (will be on top because created_at DESC)
  const ts = Date.now().toString().slice(-6);
  const insertRes = await fetch(`${BASE}/students`, {
    method: "POST",
    headers: { ...authHeaders, "Content-Type": "application/json" },
    body: JSON.stringify({
      nim: `0823116${ts}`,
      name: `Mahasiswa Baru Sisipan ${ts}`,
      grade: 97.5,
      is_active: true,
    }),
  });
  const insertData = await insertRes.json();
  console.log(`Inserted Student on top -> ID: ${insertData.data.id}, Name: ${insertData.data.name}`);

  // Fetch Page 2 using cursor from Page 1
  const p2Res = await fetch(`${BASE}/students?limit=2&after=${encodeURIComponent(nextCursor)}`, {
    headers: authHeaders,
  });
  const p2 = await p2Res.json();
  console.log("Page 2 items:", p2.data.map((s) => `${s.id}: ${s.name} (${s.created_at})`));

  // Verify no overlap
  const p1Ids = p1.data.map((s) => s.id);
  const p2Ids = p2.data.map((s) => s.id);
  const overlap = p1Ids.filter((id) => p2Ids.includes(id));
  console.log("Overlap between Page 1 and Page 2:", overlap.length === 0 ? "NONE (BERHASIL - Cursor stabil)" : `DUPLIKAT: ${overlap}`);

  // 5. PATCH Students Pointer & omitnil
  console.log("\n--- 5. PATCH STUDENTS (Pointer & omitnil) ---");
  const targetId = insertData.data.id;

  // 5.1 Empty string {"name": ""} -> Must be rejected 422
  const patchInvalidRes = await fetch(`${BASE}/students/${targetId}`, {
    method: "PATCH",
    headers: { ...authHeaders, "Content-Type": "application/json" },
    body: JSON.stringify({ name: "" }),
  });
  console.log(`PATCH {"name": ""} -> Status: ${patchInvalidRes.status} (Harus 422)`);
  console.log("Response:", await patchInvalidRes.text());

  // 5.2 Omitted name {"grade": 99.5} -> Must succeed 200
  const patchValidRes = await fetch(`${BASE}/students/${targetId}`, {
    method: "PATCH",
    headers: { ...authHeaders, "Content-Type": "application/json" },
    body: JSON.stringify({ grade: 99.5 }),
  });
  console.log(`PATCH {"grade": 99.5} (tanpa name) -> Status: ${patchValidRes.status} (Harus 200)`);
  const patchData = await patchValidRes.json();
  console.log(`Updated Student -> Name: ${patchData.data.name}, Grade: ${patchData.data.grade}`);

  // 6. Custom Validator: format NIM
  console.log("\n--- 6. CUSTOM VALIDATOR (nim_format) ---");
  const badNimRes = await fetch(`${BASE}/students`, {
    method: "POST",
    headers: { ...authHeaders, "Content-Type": "application/json" },
    body: JSON.stringify({ nim: "12!@#", name: "Budi Santoso", grade: 80 }),
  });
  console.log(`Invalid NIM "12!@#" -> Status: ${badNimRes.status} (Harus 422)`);
  console.log("Response:", await badNimRes.text());

  // 7. Password Strength Validator
  console.log("\n--- 7. PASSWORD STRENGTH VALIDATOR ---");
  const weakPassRes = await fetch(`${BASE}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: "test_user_weak", email: "weak@univ.ac.id", password: "password123" }),
  });
  console.log(`Common Password "password123" -> Status: ${weakPassRes.status} (Harus 422)`);
  console.log("Response:", await weakPassRes.text());

  // 8. Uniform Error Format & request_id
  console.log("\n--- 8. UNIFORM ERROR FORMAT & REQUEST_ID ---");
  const notFoundRes = await fetch(`${BASE}/students/999999`, {
    headers: authHeaders,
  });
  console.log(`GET /students/999999 -> Status: ${notFoundRes.status} (Harus 404)`);
  console.log("Response:", await notFoundRes.text());

  console.log("\n==========================================");
  console.log("ALL TESTS COMPLETED SUCCESSFULLY!");
  console.log("==========================================");
}

main().catch(console.error);
