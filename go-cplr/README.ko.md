# sdui-template-compiler (Go)

> English: [README.md](README.md)

[SDUI 템플릿 컴파일러](../README.ko.md)의 Go 포트이다. Go ≥ 1.22를 사용하며 의존성은 하나(`gopkg.in/yaml.v3`)이다. TypeScript 레퍼런스를 포팅했으며 [`../spec`](../spec)의 공유 적합성 스위트(conformance suite), 즉 드리프트 가드가 바이트 단위 동일성을 보장한다.

모듈 경로는 `github.com/David-Lee-dev/sdui-template-compiler/go-cplr`이다. 태그가 붙은 모듈 버전이 게시되기 전에는 체크아웃과 `replace` 지시문을 사용한다.

```go.mod
require github.com/David-Lee-dev/sdui-template-compiler/go-cplr v0.0.0
replace github.com/David-Lee-dev/sdui-template-compiler/go-cplr => ../sdui-template-compiler/go-cplr
```

## CLI

```sh
go run ./cmd/sdui-compile <sdui-root> --out <dir> [--pretty]
```

`<dir>/manifest.json`과 `<dir>/screens/<id>/<templateVersion>.json`을 기록한다. `--pretty`는 스크린 JSON의 가독성에만 영향을 주며 etag는 항상 압축 형식의 해시이다.

## 라이브러리

```go
import sdui "github.com/David-Lee-dev/sdui-template-compiler/go-cplr"

// Compiles every screen under the root eagerly — errors on any template problem.
registry, err := sdui.BuildScreenRegistry("path/to/sdui-root")
if err != nil { /* template error: diagnostic substrings fixed by the spec */ }

// Newest version threshold ≤ the client app version
// (selection rules: ../docs/integration.md).
composed, err := registry.Resolve("home", "2.3.0")
if err != nil { /* e.g. malformed app version */ }
if composed == nil { /* unknown screen — map to your 404 */ }
// composed.Template — client-ready JSON tree (ordered Value model)
// composed.Etag     — sha256[:16] of the compact serialization

template, err := registry.Get("home", "2.3.0") // just the template (nil, nil when unknown)
registry.IDs()                                 // all screen ids, in discovery order
registry.VersionsOf("home")                    // app-version thresholds, ascending
registry.ModuleOf("home")                      // screen.yaml manifest (versions, params)
```

실패는 일반 `error` 반환값이다. 메시지는 TS 및 Python 포트와 같은 진단 부분 문자열을 포함한다.

이 포트에만 해당하는 참고 사항은 두 가지이다.

- **순서가 있는 값**: 계약이 YAML 키 순서를 보존하므로 조합된 템플릿은 전용 `Value` 모델을 사용한다(`map[string]any`를 사용하지 않는다). `encoding/json`은 키를 정렬하기 때문이다.
- **직렬화**: 직접 작성한 JSON 라이터가 `JSON.stringify` 의미론(압축 바이트, JS 숫자 형식)을 재현하므로 출력과 etag가 레퍼런스와 바이트 단위로 일치한다.

## 테스트

```sh
go test ./...        # conformance suite against ../spec
```
