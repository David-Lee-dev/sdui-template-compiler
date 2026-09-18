# 템플릿 작성 가이드

> English: [template-guide.md](template-guide.md)

이 컴파일러가 빌드하는 SDUI 템플릿의 작성법을 설명하는 튜토리얼이다. 빈 디렉터리에서 시작해 컴포넌트, 토큰, 버전이 있는 스크린까지 다룬다. 정확한 규칙과 오류 메시지를 정의하는 규범 계약은 [`spec/SPEC.md`](../spec/SPEC.md)이다. 이 가이드와 명세가 다르면 명세가 우선한다. 완전하고 테스트된 예제는 [`spec/fixtures/`](../spec/fixtures)에 있다. 여기의 모든 예시는 해당 픽스처에서 가져왔거나 호환된다.

## 개념 모델

템플릿 파일에는 첫 글자로 구분되는 두 가지 어휘가 공존한다.

- **점으로 시작하는 키**(`.ref`, `.vars`, `.defaults`, `.content`, `.token`)는 **빌드 언어**이다. 컴파일러가 컴파일 시 모두 해석하므로 출력에는 하나도 남지 않는다. 점으로 시작하는 *문자열*(`.slotname`)도 빌드 언어이지만 컴포넌트의 `.content` 내부에서만 그러하며, 다른 곳에서는 일반 문자열이다.
- **밑줄로 시작하는 키와 `${...}` 문자열**(`_type`, `_children`, `_scope`, `${item.title}` 등)은 **런타임 언어**이다. 컴파일러는 이를 그대로 통과시킨다. 그 의미는 JSON을 렌더링하는 클라이언트 엔진이 정한다. Flutter 엔진은 [스타터 키트](https://github.com/David-Lee-dev/sdui-flutter-starter-kit)를 참조한다.

이 가이드는 빌드 언어를 설명한다. 예제의 런타임 키는 현실적인 페이로드를 보여 주기 위해서만 등장한다.

빌드 언어 규칙 하나는 런타임 영역에도 적용된다. 템플릿 노드 안의 `screen_id`는 거부된다. 스크린 식별 정보는 `screen.yaml`에만 속한다.

## 1. SDUI 루트

컴파일 단위는 하나의 디렉터리 트리이다.

```text
<root>/
  _tokens/<group>.yaml            # design-token maps
  _components/<name>.yaml         # shared components and fragments
  screens/<dir>/
    screen.yaml                   # screen manifest: id, versions, params
    template/<version>/_root.yaml # the screen tree for that template version
```

가장 작은 유효 루트는 스크린 하나이다.

```yaml
# screens/hello/screen.yaml
id: hello
versions:
  '1.0.0': { template: '1.0.0' }
```

```yaml
# screens/hello/template/1.0.0/_root.yaml
_type: text
value: hello, world
```

다음과 같이 컴파일한다.

```sh
sdui-compile <root> --out build/
# build/manifest.json
# build/screens/hello/1.0.0.json
```

## 2. `screen.yaml` — 스크린 매니페스트

```yaml
id: home              # authoritative screen id (non-empty)
versions:             # app-version threshold -> asset versions for that threshold
  '1.0.0': { template: '1.0.0' }
  '2.3.0': { template: '2.0.0' }
params: [id]          # optional: route/query keys forwarded to the client as root data
```

- `id`는 출력 경로(`screens/<id>/...`), 매니페스트 키, 클라이언트가 요청하는 id 등 모든 곳에서 스크린을 식별한다. 루트 전체에서 id가 중복되면 빌드가 실패하며, id와 템플릿 버전은 숫자로만 구성할 수 없다(`"10"` 같은 매니페스트 키는 JS 객체가 순서를 바꿀 수 있다).
- `versions`는 **앱 버전 임계 버전(threshold)**을 **템플릿 버전**에 매핑한다. 앱 버전이 `V`인 클라이언트는 **V 이하인 가장 최신 임계 버전**의 에셋을 받는다. 모든 임계 버전보다 낮거나 버전을 보내지 않은 클라이언트는 가장 오래된 에셋을 받는다. 위 표에서는 앱 `1.0.0`–`2.2.x`에 템플릿 `1.0.0`을, 앱 `2.3.0` 이상에 `2.0.0`을 제공한다.
- 임계 버전 키와 이에 대조하는 클라이언트 버전은 엄격한 `major.minor.patch` 형식이어야 한다(`1.0`과 `01.0.0`은 유효하지 않다). 템플릿 버전은 디렉터리와 출력 이름으로 쓰이는 불투명한 비어 있지 않은 문자열이다. semver 형식 값은 관례이지 필수 조건이 아니다. 둘 다 따옴표로 감싼다. 따옴표 없는 `1.0.0`도 문제없지만 일관성을 유지하면 YAML의 예기치 않은 동작을 피할 수 있다.
- `params`는 매니페스트에 그대로 전달되는 메타데이터이다. 엔진은 이를 사용해 라우트 쿼리 값을 스크린의 루트 데이터로 전달한다. 컴파일러는 이 값을 해석하지 않는다.

템플릿 버전은 `template/1.0.0/_root.yaml`, `template/2.0.0/_root.yaml` 같은 디렉터리이다. 새 템플릿 디렉터리를 가리키는 임계 버전 항목을 추가하면 신형 앱의 스크린만 변경할 수 있다. 구형 앱에는 이전 트리가 계속 컴파일되어 제공된다.

> **제자리 수정과 버전 올리기.** `screen.yaml`을 건드리지 않고 `template/1.0.0/**`을 수정할 수 있다. 버전 문자열이 바뀌지 않아도 출력 *콘텐츠 해시*(etag)는 바뀌므로 etag로 재검증하는 클라이언트는 업데이트를 받는다. 변경 사항이 구형 앱에 전달되면 안 되는 경우 템플릿 버전을 올린다.

## 3. 디자인 토큰 — `.token`

토큰 그룹은 `_tokens/` 아래의 YAML 맵이다.

```yaml
# _tokens/color.yaml
primary: '#5B8CFF'
surface:
  card: '#1A1D24'
```

그룹 이름을 먼저 쓰고 키를 이어 붙인 점 경로로 참조한다.

```yaml
style:
  color: { .token: color.primary }
background: { .token: color.surface.card }
```

규칙은 다음과 같다.

- 경로에는 **두 개 이상의 세그먼트**가 있어야 한다(`group.key`; 더 깊어도 된다).
- `.token` 노드에는 **형제 키가 없어야 한다**. 노드 자체가 값이다.
- 조회한 값은 노드를 **그대로** 대체한다. 문자열, 숫자, 맵 등 토큰 파일이 담은 모든 값이 가능하다. 빌드 키가 있는지 다시 탐색하지 않는다. 토큰 파일에는 일반 값만 담으며 `.ref`나 추가 `.token`을 담지 않는다.
- 알 수 없는 그룹, 알 수 없는 경로, 맵이 아닌 값 내부로의 순회는 모두 문제가 된 경로를 메시지에 포함하며 빌드를 실패시킨다.

## 4. 컴포넌트와 프래그먼트 — `.ref`

`.ref`는 다른 파일을 포함한다. 경로 형식은 두 가지이다.

- `/name` — 절대 경로: `<root>/_components/name.yaml`로 해석한다. 하위 디렉터리도 사용할 수 있다(`/nested/wrap`).
- `name`, `./name`, `../x` — **참조하는 파일**의 디렉터리를 기준으로 하는 상대 경로. `_root.yaml` 옆에 스크린 전용 프래그먼트를 둘 때 유용하다.

`.ref` 노드에는 `_type`을 선언할 수 없다. 포함하는 내용이 노드를 완전히 대체한다.

### 프래그먼트 — 인라인 포함

`.vars`가 **없는** 컴포넌트 파일은 프래그먼트이다. 내용이 그 자리에 확장되며 인자를 받지 않는다. 인자를 전달하면 빌드 오류가 발생한다.

```yaml
# screens/home/template/1.0.0/_state.yaml
items: null
```

```yaml
# _root.yaml
_scope:
  _state: { .ref: _state }     # relative ref to the sibling file
```

큰 스크린은 프래그먼트를 사용해 가독성을 유지한다. 상태, 액션, 섹션을 형제 파일로 나누고 `_root.yaml`에서 `.ref`로 참조한다.

### 컴포넌트 — 선언된 매개변수

`.vars`가 **있는** 파일은 매개변수화된 컴포넌트이다.

```yaml
# _components/badge.yaml
.vars: [label, color, icon]          # every accepted arg, declared up front
.defaults:
  color: { .token: color.primary }   # used when the caller omits the arg
.content:                            # the tree that replaces the .ref node
  _type: row
  _children:
    - .icon                          # slot: substituted with the `icon` arg
    - _type: text
      value: .label
      style: { color: .color }
```

인자는 `.ref`의 형제 키로 전달한다.

```yaml
- { .ref: /badge, label: hello }                     # default color, no icon
- { .ref: /badge, label: alert, color: '#FF0000',
    icon: { _type: icon, name: star } }
```

규칙은 다음과 같다.

- `.vars`에는 **중복되지 않은 비어 있지 않은** 이름을 나열하며 `.content`는 필수이다.
- 호출자의 모든 인자와 `.defaults`의 모든 키는 `.vars`에 선언되어야 한다. 알 수 없는 이름은 빌드를 실패시킨다.

### 슬롯 — 인자가 들어가는 위치

`.content` 안에서 문자열 `.name`은 인자 `name`으로 대체된다.

- **제공됨**(호출자 또는 `.defaults`에서 제공) → 타입과 관계없이 값이 슬롯을 대체한다.
- **생략됨**(인자와 기본값이 모두 없음) → 슬롯을 **제거한다**. 슬롯을 값으로 갖던 맵 항목이나 목록 요소가 사라진다. 따라서 위의 첫 번째 `badge` 호출은 아이콘 없이 렌더링된다. `null` 자리 표시자가 생기는 대신 행의 자식이 하나 줄어든다.
- **배열 위치의 목록 값 인자** → 중첩하지 않고 주변 목록에 평탄하게 **펼친다**.

  ```yaml
  # _components/list.yaml
  .vars: [items]
  .content:
    _type: column
    _children:
      - { _type: text, value: header }
      - .items                       # a list arg splices: header, a, b
  ```

- 슬롯은 반드시 선언해야 한다. `.content` 안의 `.something` 문자열이 `.vars`에 없으면 빌드가 실패한다(`undeclared slot`). `./` 또는 `..`로 시작하는 문자열은 예외로서 일반 문자열로 남는다. `.ref` 값으로 사용할 때만 경로로 동작한다.

### 조합과 재귀

컴포넌트는 자유롭게 중첩할 수 있다. 인자 자체가 `.ref`, `.token`, 추가 컴포넌트를 포함할 수 있는 트리이다. 인자 하위 트리는 **호출자의** 컨텍스트에서 확장되므로 다른 `/badge` 인스턴스의 인자 안에서 컴포넌트를 자기 자신의 인자로 전달하는 경우(`{ .ref: /wrap, child: { .ref: /badge, ... } }`)는 잘못된 양성으로 판정되지 않는다. 실제 자기 재귀는 감지되어 `Circular reference: a -> b -> a`로 실패한다.

배열 안의 `.ref`가 배열로 확장되면 한 단계 평탄하게 펼친다. 행 목록을 담은 프래그먼트가 `_children`에 매끄럽게 들어간다.

## 5. YAML 방언

템플릿은 js-yaml v4가 구현하는 YAML 1.2 **코어 스키마**이며, 출력은 JSON과 호환되어야 한다.

- 불리언은 `true`/`false`뿐이다. `on`, `off`, `yes`, `no`는 **문자열**이다.
- `1.0.0`은 문자열로 남는다. 타임스탬프나 60진수로 파싱하지 않는다.
- 유한하지 않은 숫자(`.inf`, `.nan`)와 일반 객체가 아닌 값은 `YAML is not JSON-compatible: <path>`로 거부한다.
- **키 순서에는 의미가 있다.** 조합 과정에서 직렬화된 출력까지 바이트 단위로 보존되며, 따라서 etag에도 반영된다.

## 6. 출력

각 스크린과 서로 다른 각 템플릿 버전마다 다음을 생성한다.

- `screens/<id>/<templateVersion>.json` — 모든 빌드 키가 해석된 조합 트리의 압축 직렬화
- `manifest.json` — 스크린별 `versions` 맵, `params`, `etags`(템플릿 버전 → 압축 JSON의 sha256[:16])

`--pretty`는 스크린 JSON을 읽기 좋은 형식으로 만든다. etag는 항상 압축 형식에서 계산하므로 보기 좋은 출력은 etag를 바꾸지 않는다. 제공, 캐싱, 배포 등 이 파일을 활용하는 방법은 [`docs/integration.ko.md`](integration.ko.md)를 참조한다.

## 7. 마주치게 될 오류

적합성 스위트는 다음 진단 **부분 문자열**을 고정한다. 모든 포트의 메시지에 이 문자열이 포함된다. 주변 문구, 오류 타입, 경로는 포트별로 다를 수 있다. 권위 있는 목록은 [`spec/cases.yaml`](../spec/cases.yaml)의 `errors` 맵이다.

| 메시지에 포함되는 문자열 | 작성한 내용 |
|---|---|
| `Circular reference` | 자기 자신을 포함하는 컴포넌트 체인(메시지에 `a -> b -> a`처럼 경로가 표시됨) |
| `Unknown token group` | 존재하지 않는 그룹 파일을 참조하는 `.token` |
| `unknown args` | `.vars`에 선언되지 않은 호출자 인자 |
| `undeclared slot` | `.content`에 있지만 `.vars`에는 없는 `.name` 문자열 |
| `does not accept args` | 프래그먼트(`.vars` 없음)에 전달한 인자 |
| `Unresolved .ref` | 파일로 해석되지 않는 `.ref` 경로 |
| `must not have sibling keys` | 형제 키가 있는 `.token` 노드 |
| `Unsupported build key` | 빌드 언어에 없는 점으로 시작하는 키 |
| `Screen id must not be purely numeric` | `screen.yaml`의 숫자로만 된 `id` |
| `Template version must not be purely numeric` | 숫자로만 된 템플릿 버전 |

이외에도 각 포트는 같은 원칙에 따라 명세가 요구하는 추가 실패를 보고한다. 잘못된 버전에는 `Invalid semantic version: <v>`, 특수한 YAML에는 `YAML is not JSON-compatible: <path>`, 토큰 경로 순회 오류에는 실패한 경로가 포함된다. 다만 현재 적합성으로 고정된 항목은 위의 부분 문자열뿐이다.

## 전체 예제

- [`spec/fixtures/basic/`](../spec/fixtures/basic) — 실제와 유사한 두 스크린 앱. 토큰, `card` 컴포넌트, 스크린 전용 프래그먼트(`_state`, `_action`), `params` 전달을 보여 준다. `_root.yaml`에는 컴파일러가 통과시키는 엔진 언어인 런타임 어휘(`_scope`, `_loop`, `${...}`)가 빽빽하게 들어 있다.
- [`spec/fixtures/composition/`](../spec/fixtures/composition) — 기본값, 생략된 슬롯, 인자 안의 중첩 컴포넌트, 목록 펼치기, 상대 경로 프래그먼트 등 빌드 언어의 전체 흐름을 보여 준다.
- [`spec/fixtures/versioning/`](../spec/fixtures/versioning) — 스크린 하나, 템플릿 버전 세 개, 임계 버전 선택(2^53을 넘는 임의 정밀도 임계 버전 포함)을 보여 준다.
- [`spec/fixtures/numbers/`](../spec/fixtures/numbers) — 숫자 직렬화 경계 사례(지수 표기법, 정수 값인 부동소수점 수)를 보여 준다.
- [`spec/fixtures/errors/`](../spec/fixtures/errors) — 진단별로 최소한의 깨진 루트 하나를 제공한다.

각 픽스처의 `expected/` 디렉터리는 컴파일러의 정확한 출력이다. "이 YAML이 어떤 JSON이 되는가?"에 가장 빠르게 답하는 방법이다.
