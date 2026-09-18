# 포팅 가이드

> English: [porting.md](porting.md)

이 컴파일러를 다른 언어로 구현하는 방법을 설명한다. 완료 기준은 기계적으로 판정된다. **[`spec/SPEC.md`](../spec/SPEC.md)를 구현하고 공유 적합성 스위트(conformance suite)를 통과해야 한다.** 적합성 스위트는 [`spec/fixtures/`](../spec/fixtures)에 대해 [`spec/cases.yaml`](../spec/cases.yaml)을 실행한다. 이 스위트는 드리프트 가드이며, 상상 가능한 모든 입력이 아니라 대표 벡터로 계약을 검증한다. 따라서 구현 대상은 명세이고, 드리프트를 포착하는 수단은 스위트이다.

## 구현 대상

다음을 수행하는 라이브러리와 선택적 CLI를 구현한다.

1. SDUI 루트 디렉터리(`_tokens/`, `_components/`, `screens/*/screen.yaml` + `template/<v>/_root.yaml`)를 읽는다.
2. 빌드 키(`.ref` / `.vars` / `.defaults` / `.content` / `.token`)를 해석하여 스크린별, 템플릿 버전별 일반 JSON 트리로 만든다.
3. 앱 버전 임계 버전(threshold)에 따라 에셋을 선택한다(`resolve(id, appVersion)`).
4. 압축 JSON을 레퍼런스와 **바이트 단위로 동일하게** 직렬화하고 그 값에서 sha256[:16] etag를 도출한다.

규범 동작, 규칙의 경계 사례, 필수 오류 부분 문자열은 [`spec/SPEC.md`](../spec/SPEC.md)에 있다. 먼저 처음부터 끝까지 읽는다. 두 페이지 분량이다. 이 가이드는 명세에 담을 수 없는 워크플로와 포트에서 실제로 문제가 생기는 지점만 다룬다.

## 레퍼런스 구현

- [`js-cplr/`](../js-cplr) — TypeScript **레퍼런스**. 명세가 모호하면 TS 동작과 골든이 결정하며, 이에 맞춰 명세를 수정한다.
- [`py-cplr/`](../py-cplr), [`go-cplr/`](../go-cplr) — 동적 언어와 정적 언어에서 이 포팅 작업을 수행한 예제. 모듈 분할(yaml / composer / include-resolver / token-resolver / versioning / screen-manifest / registry / validator / cli)이 세 구현에서 일대일로 대응하므로 그 구조를 따른다.

## 워크플로

1. **적합성 하네스를 먼저 작성한다.** `../spec/cases.yaml`을 읽고 세 가지 케이스 유형을 구동하는 테스트 파일 하나를 만든다.
   - `golden`: `fixtures/<name>/input`을 컴파일하고 각 `screens/<id>/<v>.json`을 `expected/`와 **바이트 단위로** 비교한다. etag를 `expected/manifest.json`과 비교하고, CLI 형식의 매니페스트 객체를 다시 빌드해 보기 좋은 형식의 직렬화를 `expected/manifest.json`과 바이트 단위로 비교한다. 이는 탐색 순서도 고정한다.
   - `versioning`: 픽스처를 빌드하고 `resolve(screen, app_version)`가 각 쿼리에 대해 예상 템플릿을 선택하는지, 각 `invalid_app_versions` 항목이 오류를 발생시키는지 검증한다.
   - `errors`: `fixtures/errors/<name>/input` 아래의 각 픽스처는 목록의 부분 문자열을 **포함하는** 메시지와 함께 실패해야 한다.

   `js-cplr/test/conformance.test.ts`, `py-cplr/tests/test_conformance.py`, `go-cplr/conformance_test.go`를 참조한다. 모두 약 200줄 이하이다. 스위트는 형제 체크아웃 레이아웃(포트 디렉터리에서 `../spec`)을 전제로 한다.
2. **통과할 때까지 구현한다.** 생산적인 순서는 YAML 로딩 → 토큰 해석 → 인클루드/컴포넌트 확장 → 스크린 매니페스트 + 버전 관리 → 레지스트리 + etag이다. 각 계층을 충실히 구현하면 오류 케이스도 대체로 자연스럽게 처리된다.
3. **골든을 재생성하지 않는다.** `expected/` 파일이 계약이다. 출력이 다르면 포트가 잘못된 것이다. 또는 레퍼런스 버그를 발견한 것일 수 있는데, 이는 픽스처 수정이 아니라 명세 논의 대상이다([기여 가이드](../CONTRIBUTING.ko.md#계약-변경)).

## 포트가 실제로 실패하는 지점

아래 항목은 모두 Python 또는 Go 포트에서 실제로 문제가 되었다. 트리 순회보다 이 부분에 시간을 배정한다.

**JSON 직렬화.** 계약은 `JSON.stringify` 의미론이다. 공백 없는 압축 형식, UTF-8, 모든 변환 과정에서 **YAML 원문의 키 순서 보존**을 요구한다.

- 해당 언어의 기본 맵은 **정렬되거나 순서가 없을** 가능성이 높다. Python은 별도 처리가 필요 없었지만(`dict`가 삽입 순서를 보존), Go는 `map[string]any`와 `encoding/json`이 키를 정렬하므로 순서 전용 값 모델과 직접 작성한 JSON 라이터가 필요했다.
- 숫자는 JavaScript처럼 직렬화한다. 정수 값인 부동소수점 수는 뒤에 `.0` 없이 출력하고(`1.0` → `1`), 정수가 아닌 부동소수점 수는 왕복 가능한 최단 형식으로 출력한다. 포매터를 골든의 리터럴과 대조한다.
- 후행 줄바꿈이 없어야 하며 JSON에서 요구하는 범위 이상으로 이스케이프하지 않는다.

etag는 정확히 이 바이트의 sha256을 16자리 16진수로 절단한 값이다. 바이트가 올바르면 etag는 저절로 맞는다. 미묘하게라도 틀리면 모든 골든이 한꺼번에 실패하는데, 이는 스위트가 의도대로 작동하는 것이다.

**YAML 방언.** 계약은 **js-yaml v4가 구현하는 YAML 1.2 코어 스키마**이다. 대부분의 YAML 라이브러리는 기본적으로 YAML 1.1을 사용한다.

- `on` / `off` / `yes` / `no`는 불리언이 아니라 **문자열**로 남아야 한다(PyYAML의 기본 로더는 이를 잘못 처리하므로 Python 포트에서 리졸버를 조정한다).
- `1.0.0`은 문자열로 남는다. 타임스탬프 타입이나 60진수 숫자로 처리하지 않는다.
- 유한하지 않은 부동소수점 수와 일반 객체가 아닌 값은 `YAML is not JSON-compatible: <path>`로 거부한다. 그대로 통과시키거나 조용히 강제 변환하지 않는다.

**Semver.** 정규식으로 엄격한 `major.minor.patch`를 적용한다. `1.0`, `01.0.0`, `v1.0.0`은 모두 유효하지 않다. 사전식이 아닌 숫자로 비교한다. 버전 없음 / 빈 값 / 모든 임계 버전 미만 → **가장 오래된** 임계 버전이라는 폴백에 주의한다.

**오류 메시지.** 포트는 `cases.yaml#errors`의 부분 문자열을 **포함하는** 메시지를 던지거나 반환해야 한다. 주변 문구는 자유지만 부분 문자열은 고정이다. 메커니즘은 각 언어 관용을 따른다. TS/Python은 예외, Go는 `error` 반환을 사용한다.

**확장 컨텍스트.** 컴포넌트 인자는 대상이 활성 경로 체인에 합류하기 전에 **호출자의** 컨텍스트에서 확장한다. 그렇게 구현하지 않으면 composition 픽스처의 "다른 컴포넌트의 인자 안에 있는 컴포넌트" 케이스가 잘못된 순환 참조를 보고한다. 이 체인의 하나 차이 오류가 전형적인 composer 버그이다.

## 마무리

- 생태계에서 필요하다면 CLI를 추가한다. `sdui-compile <root> --out <dir> [--pretty]` 형식으로 `screens/<id>/<v>.json`과 `manifest.json`을 기록한다. 매니페스트는 항상 2칸 들여쓰기를 사용한 보기 좋은 형식이다.
- 공개 표면을 동일하게 제공한다. `ScreenRegistry.build` / `resolve` / `get` / `ids` / `versionsOf` / `moduleOf`를 제공하며 관용적인 대소문자 표기는 허용된다.
- 루트 README 표와 [기여 가이드](../CONTRIBUTING.ko.md)의 테스트 매트릭스에 포트와 한 줄 테스트 명령을 추가한다.
