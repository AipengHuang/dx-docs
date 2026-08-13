#!/usr/bin/env bash

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_BASE="${DIXIAN_API_BASE:-http://127.0.0.1:8080/api/v1}"
OLLAMA_BASE="${DIXIAN_OLLAMA_BASE:-http://127.0.0.1:11434}"
AUTH_FILE="${DIXIAN_AUTH_FILE:-}"
EMAIL="${DIXIAN_ADMIN_EMAIL:-aipenghuang@gmail.com}"

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    printf '缺少命令: %s\n' "$1" >&2
    exit 1
  }
}

for command_name in curl jq; do
  require_command "$command_name"
done

TOKEN="${DIXIAN_API_TOKEN:-}"
if [[ -z "$TOKEN" && -n "$AUTH_FILE" && -f "$AUTH_FILE" ]]; then
  TOKEN="$(jq -r '.token // empty' "$AUTH_FILE")"
fi

if [[ -z "$TOKEN" ]]; then
  read -r -s -p "请输入 ${EMAIL} 的密码: " PASSWORD
  printf '\n'
  LOGIN_PAYLOAD="$(jq -n --arg email "$EMAIL" --arg password "$PASSWORD" '{email:$email,password:$password}')"
  unset PASSWORD
  TOKEN="$(curl -fsS \
    -H 'Content-Type: application/json' \
    --data "$LOGIN_PAYLOAD" \
    "$API_BASE/auth/login" | jq -r '.token // empty')"
  unset LOGIN_PAYLOAD
fi

if [[ -z "$TOKEN" ]]; then
  printf '登录失败，未取得访问令牌。\n' >&2
  exit 1
fi

AUTH_HEADER="Authorization: Bearer ${TOKEN}"

api_get() {
  curl -fsS -H "$AUTH_HEADER" "$API_BASE$1"
}

api_post() {
  curl -fsS -H "$AUTH_HEADER" -H 'Content-Type: application/json' --data "$2" "$API_BASE$1"
}

MODEL_NAME="qwen3-embedding:0.6b"
MODEL_DIMENSION="$(curl -fsS \
  -H 'Content-Type: application/json' \
  --data "$(jq -n --arg model "$MODEL_NAME" '{model:$model,input:["帝显知识库向量维度检测"]}')" \
  "$OLLAMA_BASE/api/embed" | jq -r '.embeddings[0] | length')"

if [[ ! "$MODEL_DIMENSION" =~ ^[1-9][0-9]*$ ]]; then
  printf '无法检测本地向量模型维度。\n' >&2
  exit 1
fi

MODELS_JSON="$(api_get '/models')"
MODEL_ID="$(jq -r --arg name "$MODEL_NAME" '.data // [] | map(select(.name == $name and .type == "Embedding")) | first | .id // empty' <<<"$MODELS_JSON")"

if [[ -z "$MODEL_ID" ]]; then
  MODEL_PAYLOAD="$(jq -n \
    --arg name "$MODEL_NAME" \
    --argjson dimension "$MODEL_DIMENSION" \
    '{
      name:$name,
      display_name:"dixian 本地多语言向量模型",
      type:"Embedding",
      source:"local",
      description:"帝显知识库本地向量化模型，不向外部服务发送文档内容。",
      parameters:{
        base_url:"",
        api_key:"",
        provider:"ollama",
        embedding_parameters:{
          dimension:$dimension,
          truncate_prompt_tokens:0,
          supports_dimension_override:false
        }
      }
    }')"
  MODEL_ID="$(api_post '/models' "$MODEL_PAYLOAD" | jq -r '.data.id // empty')"
fi

if [[ -z "$MODEL_ID" ]]; then
  printf '无法创建或定位 Embedding 模型。\n' >&2
  exit 1
fi

ensure_kb() {
  local name="$1"
  local description="$2"
  local id
  id="$(api_get '/knowledge-bases' | jq -r --arg name "$name" '.data // [] | map(select(.name == $name)) | first | .id // empty')"
  if [[ -z "$id" ]]; then
    local payload
    payload="$(jq -n \
      --arg name "$name" \
      --arg description "$description" \
      --arg model_id "$MODEL_ID" \
      '{
        name:$name,
        description:$description,
        type:"document",
        embedding_model_id:$model_id,
        storage_provider_config:{provider:"local"},
        auto_tag_config:{enabled:false},
        indexing_strategy:{
          vector_enabled:true,
          keyword_enabled:true,
          wiki_enabled:false,
          graph_enabled:false
        },
        chunking_config:{
          chunk_size:900,
          chunk_overlap:120,
          separators:["\\n\\n","\\n","。","！","？","；",";"],
          enable_multimodal:false,
          enable_parent_child:false,
          parser_engine_rules:[
            {file_types:[".md",".txt",".html"],engine:"builtin"},
            {file_types:[".pdf",".docx"],engine:"docreader"}
          ]
        }
      }')"
    id="$(api_post '/knowledge-bases' "$payload" | jq -r '.data.id // empty')"
  fi
  if [[ -z "$id" ]]; then
    printf '无法创建或定位知识库: %s\n' "$name" >&2
    exit 1
  fi
  printf '%s' "$id"
}

knowledge_exists() {
  local kb_id="$1"
  local title="$2"
  local source="${3:-}"
  api_get "/knowledge-bases/${kb_id}/knowledge?page=1&page_size=100" | jq -e \
    --arg title "$title" \
    --arg source "$source" \
    '.data // [] | any((.title == $title) or (.file_name == $title) or (($source != "") and (.source == $source)))' \
    >/dev/null
}

upload_file() {
  local kb_id="$1"
  local relative_path="$2"
  local file_path="$PROJECT_DIR/$relative_path"
  local file_name
  file_name="$(basename "$file_path")"
  if knowledge_exists "$kb_id" "$file_name"; then
    printf '已存在，跳过: %s\n' "$file_name"
    return
  fi
  curl -fsS \
    -H "$AUTH_HEADER" \
    -F "file=@${file_path}" \
    -F 'metadata={"source":"dixian_bootstrap"}' \
    -F 'channel=api' \
    "$API_BASE/knowledge-bases/${kb_id}/knowledge/file" \
    | jq -r '"已上传: " + (.data.title // .data.file_name // "unknown")'
}

import_url() {
  local kb_id="$1"
  local url="$2"
  local title="$3"
  local file_name="${4:-}"
  if knowledge_exists "$kb_id" "$title" "$url"; then
    printf '已存在，跳过: %s\n' "$title"
    return
  fi
  local payload
  if [[ -n "$file_name" ]]; then
    payload="$(jq -n --arg url "$url" --arg title "$title" --arg file_name "$file_name" \
      '{url:$url,title:$title,file_name:$file_name,file_type:"pdf",enable_multimodel:false,channel:"api"}')"
  else
    payload="$(jq -n --arg url "$url" --arg title "$title" \
      '{url:$url,title:$title,enable_multimodel:false,channel:"api"}')"
  fi
  local response
  if ! response="$(api_post "/knowledge-bases/${kb_id}/knowledge/url" "$payload")"; then
    printf '远程资料导入失败，稍后需要人工复核: %s\n' "$title" >&2
    return 0
  fi
  jq -r '"已导入: " + (.data.title // .data.file_name // .data.source // "unknown")' <<<"$response"
}

COMPANY_KB="$(ensure_kb '帝显公司与业务' '帝显电子公司概况、LED 背光源、TPM 触显全贴合、EMS 电子制造服务及应用行业资料。')"
AUTOMOTIVE_KB="$(ensure_kb '车载显示与汽车标准' '车载显示项目、汽车电子环境可靠性、EMC、功能安全、器件可靠性与 IATF 公开资料。')"
EMS_KB="$(ensure_kb '工业电子与EMS制造' '工业控制、PCBA、整机装配、测试、供应链、IPC 电子装联与职业健康安全资料。')"
SYSTEM_KB="$(ensure_kb '质量、环境与医疗体系' 'ISO 9001、ISO 14001、ISO 13485 等管理体系的公开指南、适用性和实施边界。')"

upload_file "$COMPANY_KB" 'knowledge_sources/dixian-company-business.md'
upload_file "$COMPANY_KB" 'knowledge_sources/dixian-led-tpm.md'
upload_file "$COMPANY_KB" 'knowledge_sources/dixian-ems-quality.md'

upload_file "$AUTOMOTIVE_KB" 'knowledge_sources/automotive-standards-guide.md'
import_url "$AUTOMOTIVE_KB" \
  'https://unece.org/fileadmin/DAM/trans/main/wp29/wp29regs/2019/E-ECE-324-Add.9-Rev.6.pdf' \
  'UNECE UN Regulation No. 10 Revision 6' \
  'UNECE-UN-R10-Revision-6.pdf'
import_url "$AUTOMOTIVE_KB" \
  'https://www.aecouncil.com/Documents/AEC_Q100_Rev_J_Base_Document.pdf' \
  'AEC-Q100 Rev J Base Document' \
  'AEC-Q100-Rev-J.pdf'
import_url "$AUTOMOTIVE_KB" \
  'https://www.iatfglobaloversight.org/wp/wp-content/uploads/2025/11/IATF-16949-SIs-Nov-2025-CN.pdf' \
  'IATF 16949 认可解释 SI 中文版（2025-11）' \
  'IATF-16949-SIs-2025-11-CN.pdf'
import_url "$AUTOMOTIVE_KB" \
  'https://www.iatfglobaloversight.org/wp/wp-content/uploads/2026/07/IATF-169492016-%E5%B8%B8%E8%A7%81%E9%97%AE%E9%A2%98FAQs-2026%E5%B9%B46%E6%9C%88%E4%BF%AE%E8%AE%A2%E5%B9%B6%E9%87%8D%E6%96%B0%E5%8F%91%E5%B8%83.pdf' \
  'IATF 16949 常见问题中文版（2026-06）' \
  'IATF-16949-FAQs-2026-06-CN.pdf'

upload_file "$EMS_KB" 'knowledge_sources/dixian-ems-quality.md'
upload_file "$EMS_KB" 'knowledge_sources/industrial-ems-standards-guide.md'
import_url "$EMS_KB" \
  'https://www.electronics.org/news-release/ipc-releases-j-revisions-two-leading-standards-electronics-assembly' \
  'IPC 发布 J-STD-001J 与 IPC-A-610J 官方说明'

upload_file "$SYSTEM_KB" 'knowledge_sources/management-system-standards-guide.md'
import_url "$SYSTEM_KB" \
  'https://www.iso.org/files/live/sites/isoorg/files/store/cn/PUB100499_cn.pdf' \
  'ISO 14001:2026 公开中文宣传册' \
  'ISO-14001-2026-brochure-CN.pdf'
import_url "$SYSTEM_KB" \
  'https://www.iso.org/files/live/sites/isoorg/files/store/en/PUB100377.pdf' \
  'ISO 13485 公开英文宣传册' \
  'ISO-13485-brochure-EN.pdf'

printf '\n知识库初始化请求已提交。模型维度: %s\n' "$MODEL_DIMENSION"
api_get '/knowledge-bases' | jq -r '.data // [] | map(select(.name | startswith("帝显") or startswith("车载") or startswith("工业电子") or startswith("质量")))[] | "- \(.name): \(.knowledge_count // 0) 项，处理中 \(.processing_count // 0) 项"'

unset TOKEN AUTH_HEADER
