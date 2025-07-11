$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
$session.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
$session.Cookies.Add((New-Object System.Net.Cookie("5", "1", "/", "duckduckgo.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("dcs", "1", "/", "duckduckgo.com")))
$session.Cookies.Add((New-Object System.Net.Cookie("dcm", "3", "/", "duckduckgo.com")))

try {
    $response = Invoke-WebRequest -UseBasicParsing -Uri "https://duckduckgo.com/duckchat/v1/status" `
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
    
    Write-Host "Status Code:" $response.StatusCode
    Write-Host "VQD Header:" $response.Headers["x-vqd-hash-1"]
    Write-Host "All Headers:"
    $response.Headers | Format-Table
} catch {
    Write-Host "Error:" $_.Exception.Message
    Write-Host "Status Code:" $_.Exception.Response.StatusCode
}
