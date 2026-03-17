mod tools;

use reqwest::blocking::Client;
use serde::{Deserialize, Serialize};
use std::env;
use std::fs;
use std::path::PathBuf;
use std::time::Duration;

#[derive(Serialize, Clone)]
struct Message {
    role: String,
    content: String,
}

struct Config {
    base_url: String,
    api_key: String,
    model: String,
}

#[derive(Deserialize)]
struct ApiResponse {
    choices: Vec<Choice>,
}

#[derive(Deserialize)]
struct Choice {
    message: AssistantMessage,
}

#[derive(Deserialize)]
struct AssistantMessage {
    content: String,
}

fn load_config() -> Result<Config, String> {
    let base_url =
        first_env("LLM_BASE_URL").unwrap_or_else(|| "https://api.deepseek.com/v1".to_string());
    let api_key = first_env("LLM_API_KEY")
        .or_else(|| first_env("DEEPSEEK_API_KEY"))
        .ok_or_else(|| "请设置 LLM_API_KEY 或 DEEPSEEK_API_KEY".to_string())?;
    let model = first_env("LLM_MODEL").unwrap_or_else(|| "deepseek-chat".to_string());

    Ok(Config {
        base_url: base_url.trim_end_matches('/').to_string(),
        api_key,
        model,
    })
}

fn first_env(name: &str) -> Option<String> {
    env::var(name)
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
}

fn call_model(config: &Config, messages: &[Message]) -> Result<String, String> {
    let client = Client::builder()
        .timeout(Duration::from_secs(120))
        .build()
        .map_err(|err| err.to_string())?;

    let body = serde_json::json!({
        "model": config.model,
        "messages": messages,
    });

    let response = client
        .post(format!("{}/chat/completions", config.base_url))
        .bearer_auth(&config.api_key)
        .json(&body)
        .send()
        .map_err(|err| err.to_string())?;

    let status = response.status();
    let text = response.text().map_err(|err| err.to_string())?;
    if !status.is_success() {
        return Err(format!("模型调用失败：{} {}", status, text.trim()));
    }

    let data: ApiResponse = serde_json::from_str(&text).map_err(|err| err.to_string())?;
    let content = data
        .choices
        .first()
        .map(|choice| choice.message.content.trim().to_string())
        .filter(|content| !content.is_empty())
        .ok_or_else(|| "模型返回为空".to_string())?;

    Ok(content)
}

fn parse_reply(text: &str) -> (Option<(String, String)>, Option<String>) {
    let final_text = extract_tag(text, "final");
    let action = extract_action(text);
    (action, final_text)
}

fn extract_tag(text: &str, tag: &str) -> Option<String> {
    let start_tag = format!("<{tag}>");
    let end_tag = format!("</{tag}>");
    let start = text.find(&start_tag)? + start_tag.len();
    let end = text[start..].find(&end_tag)? + start;
    Some(text[start..end].trim().to_string())
}

fn extract_action(text: &str) -> Option<(String, String)> {
    let marker = r#"<action tool=""#;
    let start = text.find(marker)? + marker.len();
    let rest = &text[start..];
    let tool_end = rest.find('"')?;
    let tool = rest[..tool_end].trim().to_string();

    let after_tool = &rest[tool_end..];
    let body_start = after_tool.find('>')? + 1;
    let body = &after_tool[body_start..];
    let body_end = body.find("</action>")?;

    Some((tool, body[..body_end].trim().to_string()))
}

fn wrap_observation(text: &str) -> String {
    let escaped = text
        .replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;");
    format!("<observation>{escaped}</observation>")
}

fn run(question: &str, config: &Config, root: &PathBuf) -> Result<String, String> {
    let prompt = fs::read_to_string("prompt.md").map_err(|err| err.to_string())?;

    let mut history = vec![
        Message {
            role: "system".to_string(),
            content: prompt,
        },
        Message {
            role: "user".to_string(),
            content: question.to_string(),
        },
    ];

    for step in 1..=8 {
        let reply = call_model(config, &history)?;
        println!("\n[第 {step} 轮]\n{reply}");

        history.push(Message {
            role: "assistant".to_string(),
            content: reply.clone(),
        });

        let (action, final_text) = parse_reply(&reply);
        if let Some(final_text) = final_text {
            return Ok(final_text);
        }

        let (tool, input) = action.ok_or_else(|| "模型输出不符合约定".to_string())?;
        let observation = wrap_observation(&tools::call_tool(root, &tool, &input));
        println!("{observation}");

        history.push(Message {
            role: "user".to_string(),
            content: observation,
        });
    }

    Err("超过最大轮数，仍未得到最终答案".to_string())
}

fn main() {
    let question = {
        let args: Vec<String> = env::args().skip(1).collect();
        if args.is_empty() {
            "请列出当前目录，并解释这个最小 Agent 的工作流程。".to_string()
        } else {
            args.join(" ")
        }
    };

    let config = match load_config() {
        Ok(value) => value,
        Err(err) => {
            eprintln!("{err}");
            std::process::exit(1);
        }
    };

    let root = match env::current_dir() {
        Ok(value) => value,
        Err(err) => {
            eprintln!("{err}");
            std::process::exit(1);
        }
    };

    println!("问题: {question}");
    let answer = match run(&question, &config, &root) {
        Ok(value) => value,
        Err(err) => {
            eprintln!("运行失败: {err}");
            std::process::exit(1);
        }
    };

    println!("\n=== 最终回答 ===");
    println!("{answer}");
}
