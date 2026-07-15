// 与后端 User 模型对齐
export interface User {
  id: number;
  username: string;
  role: "admin" | "user";
  createdAt: string;
}

export interface AuthResult {
  access_token: string;
  refresh_token: string;
  user: User;
}
