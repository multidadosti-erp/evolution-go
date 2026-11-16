import axios from "axios";

interface VerifyGoServerParams {
  url: string;
  token: string;
}

export const verifyGoServer = async ({ url, token }: VerifyGoServerParams) => {
  try {
    await axios.get(`${url}/instance/all`, {
      headers: { apikey: token, "Cache-Control": "no-cache" },
      params: { t: Date.now() },
    });
    return true;
  } catch (error) {
    return false;
  }
};
