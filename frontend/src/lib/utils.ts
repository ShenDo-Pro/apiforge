import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

// cn 合并 Tailwind 类名并去重冲突，shadcn 组件统一入口。
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
