### Claude Code Go sdk问题

#### 问题描述

1. 我对`examples/22_status_display/main.go`进行了测试，发现了几个问题：
    1.1. `DurationMs`应当要始终有输出的，因为它代表当前会话的耗时，但是example中只有最后几次有输出
    1.2. 这个example的输出和终端内的输出格式不一样，这个输出看不懂，请最好保持一致的输出格式

example output：
```
system: hook_started
system: hook_response
system: init
generating response | astron-code-latest | turn 1
generating response | astron-code-latest | turn 2
generating response | astron-code-latest | turn 3
2
processing tool result | astron-code-latest | turn 3
7
system: api_retry | astron-code-latest | turn 3
using tool: Glob | astron-code-latest | turn 4 | tokens: 31k in / 0k out | stop: tool_use
using tool: Glob | astron-code-latest | turn 5 | tokens: 62k in / 0k out | stop: tool_use
using tool: Glob | astron-code-latest | turn 6 | tokens: 93k in / 0k out | stop: tool_use
3
processing tool result | astron-code-latest | turn 6 | tokens: 93k in / 0k out | stop: tool_use
generating response | astron-code-latest | turn 7 | tokens: 93k in / 0k out
processing tool result | astron-code-latest | turn 7 | tokens: 93k in / 0k out
generating response | astron-code-latest | turn 8 | tokens: 93k in / 0k out
2
done | astron-code-latest | turn 8 | tokens: 96k in / 0k out | $0.3961 | 2m6s

--- Final Summary ---
done | astron-code-latest | turn 8 | tokens: 96k in / 0k out | $0.3961 | 2m6s
```

claude output:
```
Processing. (4m 28s • $ 1.8k tokens)

Adding Wails service method... (12m 17s • \ 9.2k tokens • thinking)

Compacting conversation... (5m 20s • 4 2.6k tokens)

Twisting. (6m 21s • 1 3.1k tokens)
```
