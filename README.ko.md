# SDUI 템플릿 컴파일러

> English: [README.md](README.md)

서버 주도형 UI를 위한 빌드 도구이다. 공유 컴포넌트, 디자인 토큰, 앱 버전 임계 버전(threshold)을 포함한 **YAML 스크린 템플릿** 트리를 완전히 조합된 클라이언트용 **JSON과 etag 매니페스트**로 컴파일한다.

이 저장소는 하나의 공유된 실행 가능 적합성 명세를 따르는 **TypeScript**(레퍼런스), **Python**, **Go** 구현 세 가지를 제공한다. 세 구현은 동일한 입력에 대해 **바이트 단위로 동일한 출력**을 생성한다.

```text
 authoring (this compiler)                        serving (your server)        rendering (client)
┌──────────────────────────┐   sdui-compile   ┌──────────────────────┐   HTTP   ┌───────────────┐
│ <root>/                  │ ───────────────▶ │ screens/<id>/<v>.json│ ───────▶ │ SDUI engine   │
│   _tokens/*.yaml         │                  │ manifest.json        │          │ (e.g. Flutter)│
│   _components/*.yaml     │                  │  · version map       │          └───────────────┘
│   screens/<id>/          │                  │  · params            │
│     screen.yaml          │                  │  · etags             │
│     template/<v>/_root…  │                  └──────────────────────┘
└──────────────────────────┘
```

## 도입 이유

- **앱을 출시하지 않고 UI를 배포한다.** 스크린은 데이터이다. YAML을 수정하고 다시 컴파일해 JSON을 배포하면 클라이언트는 다음 로드 시 변경 사항을 받는다.
- **구조적으로 작성하고 평탄하게 제공한다.** 선언된 변수와 기본값이 있는 컴포넌트(`.ref`), 디자인 토큰(`.token`), 프래그먼트를 사용하면 반복 없이 작성할 수 있다. 컴파일러가 빌드 시 모든 것을 해석하므로 서버와 클라이언트에는 일반 JSON만 전달된다.
- **앱 버전에 맞춰 템플릿 버전을 관리한다.** 각 스크린은 앱 버전 임계 버전을 템플릿 버전에 매핑한다(`screen.yaml`). `resolve(id, appVersion)`는 구형과 신형을 막론하고 모든 클라이언트에 적합한 에셋을 선택한다.
- **서버 프레임워크가 필요 없다.** 출력은 정적 JSON과 매니페스트이므로 어떤 언어, 프레임워크, CDN에서도 제공할 수 있다. 캐싱에는 표준 etag 재검증을 사용한다.
- **언어를 선택할 수 있다.** TS, Python, Go 포트는 하나의 규범 명세([`spec/SPEC.md`](spec/SPEC.md))를 구현하며 공유 골든 픽스처 스위트가 바이트 단위 동일성을 보장한다. 이는 모든 포트가 통과 상태를 유지해야 하는 드리프트 가드이다.

## 빠른 시작

TypeScript CLI로 예제 픽스처를 컴파일한다. 어느 포트를 사용해도 플래그와 결과 바이트는 동일하다.

```sh
cd js-cplr
pnpm install && pnpm build
node dist/cli.js ../spec/fixtures/basic/input --out /tmp/sdui-out --pretty
```

스크린별, 템플릿 버전별 JSON 파일 하나와 매니페스트가 생성된다.

```text
/tmp/sdui-out/
  manifest.json                 # { home: { versions, params, etags }, ... }
  screens/home/1.0.0.json
  screens/detail/1.0.0.json
```

미리 컴파일하지 않고 프로세스 내에서 스크린을 해석할 수도 있다.

```ts
import { ScreenRegistry } from 'sdui-template-compiler';

const registry = ScreenRegistry.build('path/to/sdui-root');
const composed = registry.resolve('home', '2.3.0'); // newest threshold ≤ 2.3.0
// composed.template — client-ready JSON tree
// composed.etag     — sha256[:16] content hash for cache revalidation
```

Python과 Go도 같은 모델을 제공한다. 설치 방법과 각 언어에 맞는 예제는 [포트별 README](#포트)를 참조한다.

## 템플릿의 모습

```yaml
# screens/home/template/1.0.0/_root.yaml
_type: column
_children:
  - _type: text
    value: "Today's picks"
    style: { color: { .token: color.primary } }   # design-token lookup
  - { .ref: /card, padding: 16,                   # shared component + args
      child: { _type: text, value: hello } }
```

```yaml
# _components/card.yaml — declared variables, defaults, slot substitution
.vars: [padding, child]
.defaults: { padding: 12 }
.content:
  _type: container
  padding: .padding
  _child: .child
```

컴파일러는 **점으로 시작하는 빌드 키**(`.ref`, `.vars`, `.defaults`, `.content`, `.token`)를 관할한다. 이 키는 빌드 시 해석되어 출력에 남지 않는다. 점으로 시작하는 *문자열*은 슬롯이지만 컴포넌트의 `.content` 내부에서만 그러하다. **밑줄로 시작하는 모든 항목**(`_type`, `_scope`, `${bindings}` 등)은 그대로 통과하는 런타임 어휘이며, JSON을 렌더링하는 클라이언트 엔진에 속한다. 전체 작성 가이드는 [`docs/template-guide.ko.md`](docs/template-guide.ko.md)를 참조한다.

## 포트

| 포트 | 언어 | 라이브러리 | CLI |
|---|---|---|---|
| [`js-cplr/`](js-cplr) | TypeScript(레퍼런스) | `sdui-template-compiler` (ESM) | `sdui-compile` |
| [`py-cplr/`](py-cplr) | Python ≥ 3.11 | `sdui_template_compiler` | `sdui-compile` |
| [`go-cplr/`](go-cplr) | Go ≥ 1.22 | `github.com/David-Lee-dev/sdui-template-compiler/go-cplr` | `go run ./cmd/sdui-compile` |

아직 패키지 레지스트리에 게시된 포트는 없다. 소스에서 설치해야 한다(경로 의존성, `pip install -e`, 또는 모듈 경로에 대한 Go `replace`/`go get`). 각 포트의 README에서 방법을 설명한다.

## 적합성: "바이트 단위로 동일하다"의 의미

[`spec/SPEC.md`](spec/SPEC.md)는 규범 계약이며, [`spec/cases.yaml`](spec/cases.yaml)과 [`spec/fixtures/`](spec/fixtures)는 그 실행 가능 형식이다. 모든 포트는 다음을 고정하는 동일한 적합성 스위트(conformance suite)를 실행한다.

- 조합된 출력 바이트: 압축 JSON, UTF-8, **YAML 원문의 키 순서 보존**, ECMAScript 숫자 직렬화(`1e-07`이 아닌 `1e-7`)
- CLI 매니페스트: 스크린 탐색 순서, 보기 좋은 형식의 직렬화, etag(압축 형식의 sha256을 16자리 16진수로 절단)
- 폴백과 잘못된 입력을 포함한 앱 버전 임계 버전 선택
- YAML 방언(js-yaml v4가 해석하는 YAML 1.2 코어)
- 오류 진단: 지원하는 실패 모드에서 모든 포트의 메시지는 같은 부분 문자열을 포함하므로 운영자는 어느 언어에서든 같은 진단을 확인한다.

이 스위트는 완전성을 증명하는 도구가 아니라 드리프트 가드이다. 대표 벡터를 통해 계약을 검증하며, 계약이 변경될 때마다 확장된다([기여 가이드](CONTRIBUTING.ko.md#계약-변경)). 스위트를 통과시키는 과정은 포팅 워크플로이기도 하다. [`docs/porting.ko.md`](docs/porting.ko.md)를 참조한다.

## 생태계

이 저장소는 세 부분으로 구성된 스택에서 작성 및 빌드를 담당한다. 각 부분은 독립적이다. 컴파일러는 엔진에 의존하지 않고 그 반대도 마찬가지이며, JSON과 매니페스트 형식에서만 접점을 갖는다.

| 저장소 | 역할 |
|---|---|
| **sdui-template-compiler** (이 저장소) | YAML → 조합된 JSON + 매니페스트 |
| [sdui-flutter-engine](https://github.com/David-Lee-dev/sdui-flutter-engine) | 컴파일된 JSON을 반응형으로 렌더링하는 Flutter 패키지 |
| [sdui-flutter-starter-kit](https://github.com/David-Lee-dev/sdui-flutter-starter-kit) | 두 요소를 모두 사용하는 실행 가능한 Flutter 앱 + 예제 서버 |

## 문서

| 독자 | 문서 |
|---|---|
| 템플릿 작성자 | [`docs/template-guide.ko.md`](docs/template-guide.ko.md) — 루트 레이아웃, 컴포넌트, 토큰, 슬롯, 버전 관리를 다루는 튜토리얼 |
| 서버/빌드 통합 담당자 | [`docs/integration.ko.md`](docs/integration.ko.md) — CLI와 라이브러리, 산출물 레이아웃, 제공 알고리즘, 캐시 무효화 |
| 포트 작성자 | [`docs/porting.ko.md`](docs/porting.ko.md) — 새 언어로 명세 구현하기 |
| 기여자 | [`CONTRIBUTING.ko.md`](CONTRIBUTING.ko.md) — 저장소 레이아웃, 테스트 명령, 계약 변경 워크플로 |
| 모든 사용자(규범 문서) | [`spec/SPEC.md`](spec/SPEC.md) — 컴파일러 계약 |

## 검증

각 포트는 동일한 `spec/` 벡터를 모두 통과해야 한다.

```sh
(cd js-cplr && pnpm install && pnpm test)
(cd py-cplr && python3 -m venv .venv && .venv/bin/pip install -e '.[dev]' \
            && .venv/bin/python -m pytest)
(cd go-cplr && go test ./...)
```

## 라이선스

[MIT](LICENSE)
