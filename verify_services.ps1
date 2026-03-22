$ErrorActionPreference = "Stop"

function Check-Health($url, $name) {
    try {
        $resp = Invoke-RestMethod -Uri "$url/health"
        Write-Host "✅ $name is Healthy: $($resp.status)" -ForegroundColor Green
    } catch {
        Write-Host "❌ $name Health Check Failed: $_" -ForegroundColor Red
    }
}

Write-Host "--- Checking Service Health ---"
Check-Health "http://localhost:8080" "Monolith"
Check-Health "http://localhost:8081" "User Service"
Check-Health "http://localhost:8082" "Order Service"
Check-Health "http://localhost:8000" "API Gateway"

Write-Host "`n--- Testing Strangler Pattern Functionality ---"

# 1. Create a new user in Monolith
$email = "test-verification-$(Get-Random)@example.com"
Write-Host "Creating new user in Monolith: $email"
$newUser = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/users" -Body (@{name="Verify User"; email=$email} | ConvertTo-Json) -ContentType "application/json"
$newId = $newUser.id
Write-Host "User created with ID: $newId"

# 2. Wait for CDC sync
Write-Host "Waiting 3 seconds for CDC sync..."
Start-Sleep -s 3

# 3. Verify in User Service (via Gateway)
Write-Host "Verifying new user via Gateway (:8000)..."
try {
    $gatewayUser = Invoke-RestMethod -Uri "http://localhost:8000/users/$newId"
    Write-Host "✅ New User Routing Success: Found $($gatewayUser.name) in User Service" -ForegroundColor Green
} catch {
    Write-Host "❌ New User Routing Failed: $_" -ForegroundColor Red
}

# 4. Verify Historical User Fallback (via Gateway)
Write-Host "Verifying Historical User (User 1) via Gateway Fallback..."
try {
    $histUser = Invoke-RestMethod -Uri "http://localhost:8000/users/1"
    Write-Host "✅ Historical User Fallback Success: Found $($histUser.name) from Monolith" -ForegroundColor Green
} catch {
    Write-Host "❌ Historical User Fallback Failed: $_" -ForegroundColor Red
}

# 5. Verify Historical Order Fallback (via Gateway)
Write-Host "Verifying Historical Order (Order 1) via Gateway Fallback..."
try {
    $histOrder = Invoke-RestMethod -Uri "http://localhost:8000/orders/1"
    Write-Host "✅ Historical Order Fallback Success: Found Order #$($histOrder.id) from Monolith" -ForegroundColor Green
} catch {
    Write-Host "❌ Historical Order Fallback Failed: $_" -ForegroundColor Red
}
