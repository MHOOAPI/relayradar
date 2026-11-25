// Shared labels for the service keys used by stations (claude / openai / …).
// Vendor keys map to the icon set in components/marks.tsx.

export const SERVICE_LABEL: Record<string, string> = {
  claude: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  deepseek: 'DeepSeek',
  xai: 'Grok',
  grok: 'Grok',
  moonshot: 'Kimi',
  kimi: 'Kimi',
  qwen: 'Qwen',
  google: 'Gemini',
  anthropic: 'Claude',
  image: '绘图',
  text: '文本',
  video: '视频',
  audio: '语音',
  embedding: '向量',
};

export const SERVICE_VENDOR: Record<string, string> = {
  claude: 'anthropic',
  openai: 'openai',
  gemini: 'google',
  deepseek: 'deepseek',
  xai: 'xai',
  grok: 'xai',
  moonshot: 'moonshot',
  kimi: 'moonshot',
  qwen: 'qwen',
  google: 'google',
  anthropic: 'anthropic',
};

export const serviceLabel = (key: string) => SERVICE_LABEL[key] ?? key;
