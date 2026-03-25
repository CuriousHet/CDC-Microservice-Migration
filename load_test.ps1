# Load Test Script for Dashboard
$baseUrl = "http://localhost:8000"
$totalRequests = 100

Write-Host "Starting load test: $totalRequests random requests to $baseUrl..."

for ($i = 1; $i -le $totalRequests; $i++) {
    $id = Get-Random -Minimum 1 -Maximum 6001
    $service = if ((Get-Random -Minimum 0 -Maximum 2) -eq 0) { "users" } else { "orders" }
    $url = "$baseUrl/$service/$id"
    
    try {
        # Using -UseBasicParsing to avoid IE engine dependencies on some systems
        $status = Invoke-RestMethod -Uri $url -Method Get
        Write-Host "[$i] SUCCESS: $url"
    } catch {
        Write-Host "[$i] FAILED: $url - $($_.Exception.Message)"
    }
    # Small sleep to avoid overwhelming the logs
    Start-Sleep -Milliseconds 50
}

Write-Host "`nLoad test complete. Check the dashboard at http://localhost:8000/dashboard"
