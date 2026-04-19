# Partilhado por scripts de TTS extras. Define CHATBOT_TTS_CACHE_ROOT (ex. F:\ChatBotTts) para NAO usar disco C:.
function Get-ChatbotTtsCachePaths {
    param(
        [Parameter(Mandatory = $true)]
        [string] $RepoRoot
    )
    $ext = [Environment]::GetEnvironmentVariable("CHATBOT_TTS_CACHE_ROOT", "Process")
    if ([string]::IsNullOrWhiteSpace($ext)) {
        $ext = [Environment]::GetEnvironmentVariable("CHATBOT_TTS_CACHE_ROOT", "User")
    }
    if ([string]::IsNullOrWhiteSpace($ext)) {
        $base = Join-Path $RepoRoot "services\tts-extras"
        return [pscustomobject]@{
            Mode       = "repo"
            HfHome     = Join-Path $base "hf-cache"
            FishS2Pro  = Join-Path $RepoRoot "services\tts-extras\vendor\fish-speech\checkpoints\s2-pro"
            CosyModels = Join-Path $base "pretrained_models"
        }
    }
    $r = $ext.Trim().TrimEnd('\', '/')
    return [pscustomobject]@{
        Mode       = "external"
        HfHome     = Join-Path $r "hf-cache"
        FishS2Pro  = Join-Path $r "fish-speech-s2-pro"
        CosyModels = Join-Path $r "cosyvoice-pretrained_models"
    }
}
