# Gera WAV/MP3 em voice-samples/ a partir de voice-samples/PHRASE_PT.txt (UTF-8).
# Kokoro: POST /v1/audio/speech (compat. OpenAI). OmniVoice: idem (sem response_format mp3).
param(
    [string] $RepoRoot = "",
    [string] $KokoroBaseUrl = "http://127.0.0.1:8880",
    [string] $OmniBaseUrl = "http://127.0.0.1:8000",
    [switch] $SkipKokoro,
    [switch] $SkipOmni
)

$ErrorActionPreference = "Stop"

function Write-JsonUtf8NoBom {
    param([string] $Path, [hashtable] $Obj)
    $json = ($Obj | ConvertTo-Json -Compress -Depth 6)
    [System.IO.File]::WriteAllText($Path, $json, [System.Text.UTF8Encoding]::new($false))
}

function Invoke-SpeechToFile {
    param(
        [string] $Uri,
        [string] $BodyPath,
        [string] $OutPath
    )
    $code = curl.exe -s -o $OutPath -w "%{http_code}" -X POST $Uri `
        -H "Content-Type: application/json; charset=utf-8" `
        --data-binary "@$BodyPath"
    if ($code -ne "200") {
        if (Test-Path $OutPath) { Remove-Item $OutPath -Force -ErrorAction SilentlyContinue }
        Write-Warning "HTTP $code para $OutPath"
        $err = curl.exe -s -D - -X POST $Uri -H "Content-Type: application/json; charset=utf-8" --data-binary "@$BodyPath"
        Write-Host $err
        return $false
    }
    $len = (Get-Item $OutPath).Length
    if ($len -lt 100) {
        Write-Warning "Ficheiro muito pequeno ($len bytes): $OutPath"
        return $false
    }
    Write-Host "OK -> $OutPath ($len bytes)"
    return $true
}

if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
}

$phrasePath = Join-Path $RepoRoot "voice-samples\PHRASE_PT.txt"
if (-not (Test-Path $phrasePath)) {
    Write-Error "Ficheiro em falta: $phrasePath"
}

$phrase = [System.IO.File]::ReadAllText($phrasePath, [System.Text.UTF8Encoding]::new($false)).Trim()
if ([string]::IsNullOrWhiteSpace($phrase)) {
    Write-Error "PHRASE_PT.txt está vazio."
}

$tmpDir = Join-Path $env:TEMP "chatbot-voice-samples"
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

$kokoroDir = Join-Path $RepoRoot "voice-samples\kokoro"
$omniDir = Join-Path $RepoRoot "voice-samples\omnivoice"
New-Item -ItemType Directory -Force -Path $kokoroDir | Out-Null
New-Item -ItemType Directory -Force -Path $omniDir | Out-Null

$kokoroUri = ($KokoroBaseUrl.TrimEnd("/")) + "/v1/audio/speech"
$omniUri = ($OmniBaseUrl.TrimEnd("/")) + "/v1/audio/speech"

if (-not $SkipKokoro) {
    # Kokoro-82M: secção Brazilian Portuguese (pf_dora, pm_alex, pm_santa).
    $kokoroVoices = @("pf_dora", "pm_alex", "pm_santa")
    foreach ($v in $kokoroVoices) {
        $bodyPath = Join-Path $tmpDir "kokoro-$v.json"
        $written = $false
        foreach ($try in @(
                @{ fmt = "mp3"; ext = "mp3" },
                @{ fmt = "wav"; ext = "wav" },
                @{ fmt = $null; ext = "wav" }
            )) {
            $h = @{
                model = "kokoro"
                input = $phrase
                voice = $v
            }
            if ($null -ne $try.fmt) { $h.response_format = $try.fmt }
            Write-JsonUtf8NoBom -Path $bodyPath -Obj $h
            $out = Join-Path $kokoroDir "$v.$($try.ext)"
            if (Invoke-SpeechToFile -Uri $kokoroUri -BodyPath $bodyPath -OutPath $out) {
                $written = $true
                break
            }
        }
        if (-not $written) {
            Write-Warning "Kokoro falhou para voz $v após mp3/wav/sem format."
        }
    }
}

if (-not $SkipOmni) {
    $bodyAuto = Join-Path $tmpDir "omni-auto.json"
    Write-JsonUtf8NoBom -Path $bodyAuto -Obj @{
        model = "tts-1"
        input = $phrase
        voice = "auto"
    }
    Invoke-SpeechToFile -Uri $omniUri -BodyPath $bodyAuto -OutPath (Join-Path $omniDir "auto.wav") | Out-Null

    $bodyDesign = Join-Path $tmpDir "omni-design.json"
    Write-JsonUtf8NoBom -Path $bodyDesign -Obj @{
        model    = "tts-1"
        input    = $phrase
        voice    = "design:female, young adult, portuguese accent, low pitch"
        speed    = 0.9
        num_step = 48
    }
    Invoke-SpeechToFile -Uri $omniUri -BodyPath $bodyDesign -OutPath (Join-Path $omniDir "design_female_defaults.wav") | Out-Null

    $cloneId = "atendimento_br"
    $cloneDir = Join-Path $RepoRoot "services\omnivoice-server\voice-profiles\$cloneId"
    $refWav = Join-Path $cloneDir "ref_audio.wav"
    if ((Test-Path $refWav)) {
        $bodyClone = Join-Path $tmpDir "omni-clone.json"
        Write-JsonUtf8NoBom -Path $bodyClone -Obj @{
            model = "tts-1"
            input = $phrase
            voice = "clone:$cloneId"
        }
        Invoke-SpeechToFile -Uri $omniUri -BodyPath $bodyClone -OutPath (Join-Path $omniDir "clone_${cloneId}.wav") | Out-Null
    }
    else {
        Write-Host "(Skip clone:atendimento_br - sem $refWav)"
    }
}

Write-Host "Feito. Ouve em voice-samples\kokoro e voice-samples\omnivoice"
