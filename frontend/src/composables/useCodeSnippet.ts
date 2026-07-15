import { resolveTemplate } from "@/lib/vars";
import { useEnvironmentStore } from "@/stores/environment";

// 由当前请求生成多语言调用代码片段（纯前端，零后端依赖）。
// 生成前会用活动环境/集合/全局变量做一次 {{var}} 替换，复制即可直接运行。

export interface SnippetReq {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: string;
}

export interface Snippet {
  lang: string;
  label: string;
  code: string;
}

// 把字符串包成 shell 单引号字面量（内部 ' 转义为 '\''）
function sh(s: string): string {
  return `'${s.replace(/'/g, `'\\''`)}'`;
}

// JSON 双引号字面量：跨语言安全（JS/Python/Go 通用）
function jq(s: string): string {
  return JSON.stringify(s);
}

export function generateSnippets(raw: SnippetReq): Snippet[] {
  const envStore = useEnvironmentStore();
  const vars = envStore.mergedVars;
  const req: SnippetReq = {
    method: raw.method || "GET",
    url: resolveTemplate(raw.url, vars),
    headers: Object.fromEntries(
      Object.entries(raw.headers).map(([k, v]) => [resolveTemplate(k, vars), resolveTemplate(v, vars)])
    ),
    body: resolveTemplate(raw.body, vars),
  };
  const { method, url, headers, body } = req;
  const headerEntries = Object.entries(headers).filter(([k]) => k.trim());
  const hasBody = body && !/^(GET|HEAD|DELETE)$/i.test(method);

  const curlHeaders = headerEntries.map(([k, v]) => `-H ${sh(`${k}: ${v}`)}`).join(" \\\n  ");
  const curl =
    `curl -X ${method} ${sh(url)} \\\n  ${curlHeaders}` +
    (hasBody ? ` \\\n  -d ${sh(body)}` : "");
  // 去掉尾部多余的换行反斜杠
  const curlFinal = curl.replace(/ \\\n  $/, "");

  const pyHeaders = headerEntries.map(([k, v]) => `    ${jq(k)}: ${jq(v)},`).join("\n");
  const py =
    `import requests\n\n` +
    `url = ${jq(url)}\n` +
    `headers = {\n${pyHeaders}\n}\n` +
    (hasBody ? `payload = ${jq(body)}\n` : "") +
    `response = requests.request(${jq(method)}, url${hasBody ? ", data=payload" : ""}, headers=headers)\n\n` +
    `print(response.status_code)\nprint(response.text)`;

  const jsHeaders = headerEntries.map(([k, v]) => `    ${jq(k)}: ${jq(v)},`).join("\n");
  const node =
    `const url = ${jq(url)};\n` +
    `const options = {\n  method: ${jq(method)},\n  headers: {\n${jsHeaders}\n  }${hasBody ? `,\n  body: ${jq(body)}` : ""}\n};\n\n` +
    `fetch(url, options)\n  .then((r) => r.text())\n  .then((t) => console.log(t))\n  .catch((e) => console.error(e));`;

  const goHeaders = headerEntries.map(([k, v]) => `  req.Header.Add(${jq(k)}, ${jq(v)})`).join("\n");
  const go =
    `package main\n\n` +
    `import (\n  "fmt"\n  "io"\n  "net/http"\n  "strings"\n)\n\n` +
    `func main() {\n` +
    `  url := ${jq(url)}\n  method := ${jq(method)}\n` +
    (hasBody ? `  payload := strings.NewReader(${jq(body)})\n\n` : "\n") +
    `  client := &http.Client{}\n` +
    `  req, err := http.NewRequest(method, url, ${hasBody ? "payload" : "nil"})\n  if err != nil { panic(err) }\n` +
    (goHeaders ? `${goHeaders}\n` : "") +
    `  resp, err := client.Do(req)\n  if err != nil { panic(err) }\n  defer resp.Body.Close()\n` +
    `  body, _ := io.ReadAll(resp.Body)\n  fmt.Println(string(body))\n}`;

  const xhrHeaders = headerEntries.map(([k, v]) => `xhr.setRequestHeader(${jq(k)}, ${jq(v)});`).join("\n  ");
  const xhr =
    `const xhr = new XMLHttpRequest();\n` +
    `xhr.open(${jq(method)}, ${jq(url)});\n` +
    (xhrHeaders ? `  ${xhrHeaders}\n` : "") +
    `xhr.onload = () => console.log(xhr.responseText);\n` +
    `xhr.onerror = (e) => console.error(e);\n` +
    `xhr.send(${hasBody ? jq(body) : "null"});`;

  return [
    { lang: "shell", label: "cURL", code: curlFinal },
    { lang: "python", label: "Python", code: py },
    { lang: "javascript", label: "Node / Fetch", code: node },
    { lang: "go", label: "Go", code: go },
    { lang: "xhr", label: "JavaScript (XHR)", code: xhr },
  ];
}
