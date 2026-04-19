package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"wa-saas/backend/internal/config"
	"wa-saas/backend/internal/cryptoagent"
	"wa-saas/backend/internal/model"
)

// DefaultGeminiTTSModel modelo TTS Gemini (Speech) v1beta.
const DefaultGeminiTTSModel = "gemini-2.5-flash-preview-tts"

// geminiTTSVoiceCanonical mapeia nomes case-insensitive para o valor da API.
var geminiTTSVoiceCanonical = map[string]string{
	"kore": "Kore", "puck": "Puck", "charon": "Charon", "zephyr": "Zephyr",
	"aoede": "Aoede", "fenrir": "Fenrir", "leda": "Leda", "orus": "Orus",
}

// GeminiTTSWhatsAppInstructionDefault — ritmo com tags (como amostra venda-casa); texto sem tags continua natural.
const GeminiTTSWhatsAppInstructionDefault = "Segue estas etiquetas de interpretação (não digas a palavra \"etiqueta\" nem o nome das tags em voz alta; são só para ritmo e expressão):\n" +
	"- [PAUSA]: silêncio curto, como quem respira no microfone.\n" +
	"- [HESITA]: hesitação natural antes do próximo trecho.\n" +
	"- [GAGUEJA]: um único tropeço breve no sítio marcado (não inventes mais gaguejos noutros sítios).\n\n" +
	"Mensagem de voz pelo WhatsApp em português do Brasil: tom conversacional, caloroso e credível. " +
	"Se o texto não tiver estas etiquetas, mantém o mesmo estilo natural com pequenas pausas entre ideias, sem soar robotizado.\n\n" +
	"Lê o texto seguinte:\n"

// MaxGeminiTTSInputRunes limite conservador (API Gemini TTS).
const MaxGeminiTTSInputRunes = 8000

var gemini429RetryRE = regexp.MustCompile(`(?i)retry in ([\d.]+)\s*s`)

// EffectiveGeminiTTSModel: coluna openai_tts_model (reutilizada), depois env, depois default.
func EffectiveGeminiTTSModel(agent *model.AIAgent, cfg *config.Config) string {
	if agent != nil {
		if m := strings.TrimSpace(agent.OpenAITTSModel); m != "" {
			return m
		}
	}
	if cfg != nil {
		if m := strings.TrimSpace(cfg.GeminiTTSModel); m != "" {
			return m
		}
	}
	return DefaultGeminiTTSModel
}

// GeminiTTSInstructionPrefix texto antes do conteúdo a ler (env GEMINI_TTS_INSTRUCTION ou default).
func GeminiTTSInstructionPrefix(cfg *config.Config) string {
	if cfg != nil {
		if s := strings.TrimSpace(cfg.GeminiTTSInstruction); s != "" {
			return strings.TrimRight(s, "\n") + "\n\n"
		}
	}
	return GeminiTTSWhatsAppInstructionDefault
}

// CanonicalGeminiTTSVoice normaliza para nome aceite pela API.
func CanonicalGeminiTTSVoice(voice string) string {
	v := strings.TrimSpace(voice)
	if v == "" {
		return "Kore"
	}
	if c, ok := geminiTTSVoiceCanonical[strings.ToLower(v)]; ok {
		return c
	}
	r := []rune(strings.ToLower(v))
	if len(r) == 0 {
		return "Kore"
	}
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// ResolveGeminiTTSAPIKey: chave dedicada do agente, ou chave do LLM Gemini, ou GEMINI_API_KEY no servidor.
func ResolveGeminiTTSAPIKey(encKey string, cfg *config.Config, a *model.AIAgent) (string, error) {
	if a == nil {
		return "", fmt.Errorf("agente nil")
	}
	if strings.TrimSpace(a.GeminiTTSAPICipher) != "" {
		return cryptoagent.Decrypt(a.GeminiTTSAPICipher, encKey)
	}
	if strings.ToLower(strings.TrimSpace(a.Provider)) == "gemini" && strings.TrimSpace(a.APIKeyCipher) != "" {
		return cryptoagent.Decrypt(a.APIKeyCipher, encKey)
	}
	if cfg != nil {
		if k := strings.TrimSpace(cfg.GeminiAPIKey); k != "" {
			return k, nil
		}
	}
	return "", fmt.Errorf("sem chave Google para TTS Gemini (defina gemini_tts_api_key no agente, ou use LLM Gemini com api_key, ou GEMINI_API_KEY no servidor)")
}

// SynthGeminiTTS gera WAV mono (PCM16 ou ficheiro WAV embutido) via Gemini Speech API.
func SynthGeminiTTS(ctx context.Context, apiKey, model, voiceName, instructionPrefix, phrase string) ([]byte, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("gemini tts: api key em falta")
	}
	if model == "" {
		model = DefaultGeminiTTSModel
	}
	voiceName = CanonicalGeminiTTSVoice(voiceName)
	prefix := strings.TrimSpace(instructionPrefix)
	if prefix == "" {
		prefix = GeminiTTSWhatsAppInstructionDefault
	} else if !strings.HasSuffix(prefix, "\n") {
		prefix += "\n"
	}
	fullText := strings.TrimRight(prefix, "\n") + "\n\n" + strings.TrimSpace(phrase)

	u := "https://generativelanguage.googleapis.com/v1beta/models/" +
		url.PathEscape(model) + ":generateContent?key=" + url.QueryEscape(apiKey)

	body := map[string]interface{}{
		"contents": []interface{}{
			map[string]interface{}{
				"role": "user",
				"parts": []interface{}{
					map[string]interface{}{"text": fullText},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseModalities": []string{"AUDIO"},
			"speechConfig": map[string]interface{}{
				"voiceConfig": map[string]interface{}{
					"prebuiltVoiceConfig": map[string]interface{}{
						"voiceName": voiceName,
					},
				},
			},
		},
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt < 6; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(rawBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			pcm, wavBytes, sr, werr := extractGeminiTTSAudioFromJSON(raw)
			if werr != nil {
				return nil, werr
			}
			if len(wavBytes) > 0 {
				return wavBytes, nil
			}
			return wrapPCM16LEMonoWav(pcm, sr)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 5 {
			wait := parseGemini429RetrySeconds(raw)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
			lastErr = fmt.Errorf("gemini tts: HTTP %d", resp.StatusCode)
			continue
		}
		return nil, fmt.Errorf("gemini tts: HTTP %d: %s", resp.StatusCode, truncateErrBody(raw))
	}
	if lastErr != nil {
		return nil, fmt.Errorf("gemini tts: %w", lastErr)
	}
	return nil, fmt.Errorf("gemini tts: falha após tentativas")
}

func parseGemini429RetrySeconds(raw []byte) time.Duration {
	m := gemini429RetryRE.FindSubmatch(raw)
	if len(m) < 2 {
		return 18 * time.Second
	}
	sec, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil || sec <= 0 {
		return 18 * time.Second
	}
	d := time.Duration(sec*float64(time.Second)) + time.Second
	if d > 90*time.Second {
		d = 90 * time.Second
	}
	return d
}

func extractGeminiTTSAudioFromJSON(raw []byte) (pcm []byte, wav []byte, sampleRate int, err error) {
	var root map[string]interface{}
	if err = json.Unmarshal(raw, &root); err != nil {
		return nil, nil, 0, fmt.Errorf("gemini tts: JSON inválido: %w", err)
	}
	if errObj, ok := root["error"].(map[string]interface{}); ok {
		msg, _ := errObj["message"].(string)
		if msg == "" {
			msg = fmt.Sprintf("%v", errObj)
		}
		return nil, nil, 0, fmt.Errorf("gemini tts: %s", msg)
	}
	cands, _ := root["candidates"].([]interface{})
	for _, ci := range cands {
		cand, _ := ci.(map[string]interface{})
		content, _ := cand["content"].(map[string]interface{})
		parts, _ := content["parts"].([]interface{})
		for _, pi := range parts {
			part, _ := pi.(map[string]interface{})
			inline, _ := part["inlineData"].(map[string]interface{})
			if inline == nil {
				inline, _ = part["inline_data"].(map[string]interface{})
			}
			if inline == nil {
				continue
			}
			b64, _ := inline["data"].(string)
			if b64 == "" {
				continue
			}
			rawAudio, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("gemini tts: base64: %w", err)
			}
			mime, _ := inline["mimeType"].(string)
			if mime == "" {
				mime, _ = inline["mime_type"].(string)
			}
			mime = strings.ToLower(strings.TrimSpace(mime))
			sr := sampleRateFromMIME(mime)
			if strings.Contains(mime, "wav") && len(rawAudio) >= 12 && string(rawAudio[:4]) == "RIFF" {
				return nil, rawAudio, sr, nil
			}
			if strings.Contains(mime, "l16") || strings.Contains(mime, "pcm") {
				return rawAudio, nil, sr, nil
			}
			// fallback: PCM16
			return rawAudio, nil, sr, nil
		}
	}
	return nil, nil, 0, fmt.Errorf("gemini tts: resposta sem áudio")
}

func sampleRateFromMIME(mime string) int {
	re := regexp.MustCompile(`(?i)rate=(\d+)`)
	m := re.FindStringSubmatch(mime)
	if len(m) >= 2 {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n
		}
	}
	return 24000
}

func wrapPCM16LEMonoWav(pcm []byte, sampleRate int) ([]byte, error) {
	if sampleRate <= 0 {
		sampleRate = 24000
	}
	if len(pcm)%2 != 0 {
		return nil, fmt.Errorf("gemini tts: PCM com número ímpar de bytes")
	}
	dataSize := uint32(len(pcm))
	byteRate := uint32(sampleRate * 2)
	blockAlign := uint16(2)
	bitsPerSample := uint16(16)

	buf := new(bytes.Buffer)
	_, _ = buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+len(pcm)))
	_, _ = buf.WriteString("WAVEfmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16)) // PCM fmt chunk size
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // audio format PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // channels
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, byteRate)
	_ = binary.Write(buf, binary.LittleEndian, blockAlign)
	_ = binary.Write(buf, binary.LittleEndian, bitsPerSample)
	_, _ = buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, dataSize)
	_, _ = buf.Write(pcm)
	return buf.Bytes(), nil
}
