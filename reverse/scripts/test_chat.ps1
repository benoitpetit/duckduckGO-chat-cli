$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
$session.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
$session.Cookies.Add((New-Object System.Net.Cookie("5", "1", "/", "duckduckgo.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("dcs", "1", "/", "duckduckgo.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("dcm", "3", "/", "duckduckgo.com")))

# Get VQD first - with updated headers matching web requests
try {
    $statusResponse = Invoke-WebRequest -UseBasicParsing -Uri "https://duckduckgo.com/duckchat/v1/status" `
    -WebSession $session `
    -Headers @{
        "authority"="duckduckgo.com"
        "method"="GET"
        "path"="/duckchat/v1/status"
        "scheme"="https"
        "accept"="*/*"
        "accept-encoding"="gzip, deflate, br, zstd"
        "accept-language"="fr-FR,fr;q=0.6"
        "cache-control"="no-store"
        "dnt"="1"
        "priority"="u=1, i"
        "referer"="https://duckduckgo.com/"
        "sec-ch-ua"="`"Not)A;Brand`";v=`"8`", `"Chromium`";v=`"138`", `"Brave`";v=`"138`""
        "sec-ch-ua-mobile"="?0"
        "sec-ch-ua-platform"="`"Windows`""
        "sec-fetch-dest"="empty"
        "sec-fetch-mode"="cors"
        "sec-fetch-site"="same-origin"
        "sec-gpc"="1"
        "x-vqd-accept"="1"
    }
    
    $vqd = $statusResponse.Headers["x-vqd-hash-1"]
    Write-Host "Got VQD:" $vqd.Substring(0, 50) + "..."
    
    # Test chat request - with ALL required headers from web request
    $body = @{
        "model" = "gpt-4o-mini"
        "metadata" = @{
            "toolChoice" = @{
                "NewsSearch" = $false
                "VideosSearch" = $false
                "LocalSearch" = $false
                "WeatherForecast" = $false
            }
        }
        "messages" = @(
            @{
                "role" = "user"
                "content" = "salut :)"
            }
        )
        "canUseTools" = $true
        "canUseApproxLocation" = $true
    } | ConvertTo-Json -Depth 10
    
    Write-Host "Sending chat request with all required headers..."
    $response = Invoke-WebRequest -UseBasicParsing -Uri "https://duckduckgo.com/duckchat/v1/chat" `
    -Method "POST" `
    -WebSession $session `
    -Headers @{
        "authority"="duckduckgo.com"
        "method"="POST"
        "path"="/duckchat/v1/chat"
        "scheme"="https"
        "accept"="text/event-stream"
        "accept-encoding"="gzip, deflate, br, zstd"
        "accept-language"="fr-FR,fr;q=0.6"
        "dnt"="1"
        "origin"="https://duckduckgo.com"
        "priority"="u=1, i"
        "referer"="https://duckduckgo.com/"
        "sec-ch-ua"="`"Not)A;Brand`";v=`"8`", `"Chromium`";v=`"138`", `"Brave`";v=`"138`""
        "sec-ch-ua-mobile"="?0"
        "sec-ch-ua-platform"="`"Windows`""
        "sec-fetch-dest"="empty"
        "sec-fetch-mode"="cors"
        "sec-fetch-site"="same-origin"
        "sec-gpc"="1"
        "x-fe-signals"="eyJzdGFydCI6MTc1MjE1NTc3NzQ4MCwiZXZlbnRzIjpbeyJuYW1lIjoic3RhcnROZXdDaGF0IiwiZGVsdGEiOjc1fSx7Im5hbWUiOiJyZWNlbnRDaGF0c0xpc3RJbXByZXNzaW9uIiwiZGVsdGEiOjEyNH1dLCJlbmQiOjQzNDN9"
        "x-fe-version"="serp_20250710_090702_ET-70eaca6aea2948b0bb60"
        "x-vqd-hash-1"="eyJzZXJ2ZXJfaGFzaGVzIjpbImRQSlJJTWczZnFYQXIvaStaa3c2cEpFVzEwckdTdmxJVlVkNlFsOVRGWXc9IiwiMUN3Qzg3N0Q3WXE1dzlEeTc4UjhBVi9qZVZWaUlYbmV0Q0xvckx3c01QZz0iLCJQSzc3TGc2L25weDdWQ2J2UWxsTEhBR3cyenJIVmEvQUFBRFBhQTl1ekVRPSJdLCJjbGllbnRfaGFzaGVzIjpbImxWblI0MStCMVFWZ0o4d0hhMUdBNmdxR0JoSjlWdjN5K0dISkdGekJmTGM9IiwiVS9RRUc2RE1qdEU4V2hHU1FxOUU1Z0VGNmw1SWJrNk9NVlBuY01DU1licz0iLCJ6SURsYUNvZG9JUjNwbTNSVTlWOUJXaUJkZDJqenRMODAyN0VYTHhkWll3PSJdLCJzaWduYWxzIjp7fSwibWV0YSI6eyJ2IjoiNCIsImNoYWxsZW5nZV9pZCI6ImM4M2Q0ZTc5NTU2MjJmZjU3Mzc0ZDUzOTk2ZjliMmJhZGE2ZDQxZTMzNDM1ZjVlNzMyYjFmNmZjNmQ0ZTE1NzVoOGpidCIsInRpbWVzdGFtcCI6IjE3NTIxNTU3Nzc4NjYiLCJvcmlnaW4iOiJodHRwczovL2R1Y2tkdWNrZ28uY29tIiwic3RhY2siOiJFcnJvclxuYXQgRSAoaHR0cHM6Ly9kdWNrZHVja2dvLmNvbS9kaXN0L3dwbS5jaGF0LjcwZWFjYTZhZWEyOTQ4YjBiYjYwLmpzOjE6MTQ4MjUpXG5hdCBhc3luYyBodHRwczovL2R1Y2tkdWNrZ28uY29tL2Rpc3Qvd3BtLmNoYXQuNzBlYWNhNmFlYTI5NDhiMGJiNjAuanM6MToxNjk4NSIsImR1cmF0aW9uIjoiNTgifX0="
        "x-vqd-4"=$vqd
    } `
    -Body $body `
    -ContentType "application/json"
    
    Write-Host "Chat Status Code:" $response.StatusCode
    Write-Host "Response:" $response.Content.Substring(0, [Math]::Min(500, $response.Content.Length))
    
} catch {
    Write-Host "Error:" $_.Exception.Message
    if ($_.Exception.Response) {
        Write-Host "Status Code:" $_.Exception.Response.StatusCode
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "Response Body:" $responseBody
    }
}
