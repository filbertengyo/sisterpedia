# export-issues.ps1
$Repo = "filbertengyo/sisterpedia"   # change this
$Out  = "issues"

[Console]::OutputEncoding = [Text.Encoding]::UTF8
New-Item -ItemType Directory -Force -Path $Out | Out-Null
$OutPath = (Resolve-Path $Out).Path
$utf8 = New-Object System.Text.UTF8Encoding($false)

function Fmt($d) {
    if ($d -is [datetime]) { $d.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ") } else { $d }
}

$numbers = gh issue list -R $Repo --state all --limit 10000 --assignee "@me" --json number --jq '.[].number'

foreach ($n in $numbers) {
    $i = gh issue view $n -R $Repo --json number,title,state,author,labels,createdAt,url,body,comments |
         Out-String | ConvertFrom-Json

    $slug = ($i.title.ToLower() -replace '[^a-z0-9]+', '-').Trim('-')
    if ($slug.Length -gt 60) { $slug = $slug.Substring(0, 60).TrimEnd('-') }

    $title  = $i.title -replace '"', '\"'
    $labels = ($i.labels | ForEach-Object { '"' + $_.name + '"' }) -join ', '

    $md = @"
---
number: $($i.number)
title: "$title"
state: $($i.state)
author: $($i.author.login)
labels: [$labels]
created: $(Fmt $i.createdAt)
url: $($i.url)
---

# $($i.title)

$($i.body)

"@

    if ($i.comments.Count -gt 0) {
        $md += "`n## Comments`n"
        foreach ($c in $i.comments) {
            $md += "`n### $($c.author.login) ($(Fmt $c.createdAt))`n`n$($c.body)`n"
        }
    }

    $file = Join-Path $OutPath ("{0:D4}-{1}.md" -f [int]$n, $slug)
    [IO.File]::WriteAllText($file, $md, $utf8)
    Write-Host "Saved #$n"
}