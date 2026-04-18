const TOKEN_KEY = 'chat_app_access_token';

let accessToken: string | null = null;

export function setToken(token: string | null) {
  accessToken = token;
  if (token) {
    localStorage.setItem(TOKEN_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_KEY);
  }
}

export function getToken(): string | null {
  if (!accessToken) {
    accessToken = localStorage.getItem(TOKEN_KEY);
  }
  return accessToken;
}
