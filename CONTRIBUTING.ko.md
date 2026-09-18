# 기여 가이드

> English: [CONTRIBUTING.md](CONTRIBUTING.md)

## 저장소 레이아웃

```text
js-cplr/    TypeScript port — the reference implementation (pnpm, vitest)
py-cplr/    Python port (≥3.11, pytest)
go-cplr/    Go port (≥1.22)
spec/       The contract: SPEC.md (normative) + cases.yaml + fixtures/ (executable)
docs/       Guides: template authoring, integration, porting
```

포트들은 의도적으로 병렬 구조를 갖는다. 각각 같은 모듈 분할(yaml / composer / include-resolver / token-resolver / versioning / screen-manifest / registry / validator / cli)을 사용하고 `../spec`의 동일한 적합성 스위트(conformance suite)를 실행한다. 포트 간 공유 코드는 없으며 명세만이 유일한 결합 지점이다.

## 설정 및 테스트

각 포트는 독립적이다. 각자의 디렉터리에서 스위트를 실행한다.

```sh
(cd js-cplr && pnpm install && pnpm test)        # + pnpm typecheck
(cd py-cplr && python3 -m venv .venv && .venv/bin/pip install -e '.[dev]' \
            && .venv/bin/python -m pytest)
(cd go-cplr && go test ./...)
```

**세** 스위트를 모두 통과해야 변경이 완료된다. 아직 CI가 없으므로 푸시하기 전에 로컬에서 매트릭스를 실행한다.

## 변경 워크플로

변경 유형에 따라 수정해야 하는 대상이 달라진다.

### 포트 내부 변경(동작 변경 없음)

한 포트 내부의 리팩터링, 성능 개선, 문서, 관용적 코드 정리는 해당 포트 스위트를 통과한 일반 PR로 처리한다. 직렬화를 건드렸다면 다른 스위트도 실행한다. "동작 변경 없음"이 바로 골든이 검증하는 내용이다.

### 계약 변경

컴파일러가 받아들이거나 생성하거나 거부하는 대상을 바꾸는 모든 변경은 **명세 변경**이며, 순서대로 다음 다섯 단계를 하나의 단위로 거친다.

1. **`spec/SPEC.md`** — 규칙을 작성한다. 한두 문장의 규범 문장으로 표현할 수 없다면 설계가 끝나지 않은 것이다.
2. **`spec/cases.yaml` + `spec/fixtures/`** — 골든 픽스처, 버전 관리 쿼리, 필수 메시지 부분 문자열을 포함한 오류 케이스 중 하나로 실행 가능 형식을 추가하거나 갱신한다. 모든 새 규범 규칙에는 구현 없이는 실패하는 벡터가 필요하다. 스위트는 다루는 항목만 보호하므로 다루지 않은 규칙은 보호되지 않은 규칙이다.
3. **`js-cplr/`** — 레퍼런스에 구현한다.
4. **`py-cplr/`, `go-cplr/`** — 포팅한다.
5. **세 스위트를 모두 통과한다.**

명세와 다른 포트 없이 한 포트의 동작만 바꾸는 PR은 정의상 미완성이다. 스위트가 이를 드러낸다.

### 골든 파일은 재생성하지 않고 리뷰한다

포트 출력으로 `expected/`를 일괄 재생성하여 커밋하지 않는다. 그렇게 하면 버그를 포함한 포트의 현재 동작을 그대로 승인하게 된다. 골든을 변경해야 할 때는 다음을 따른다.

- **명세**에서 기대 바이트를 도출한다. 직접 작성하거나 리뷰를 거친 일회성 스크립트를 사용한다.
- 코드와 같은 주의로 골든 diff를 리뷰한다. 키 순서, 숫자 형식, etag가 모두 계약이다.
- 콘텐츠가 바뀌면 etag도 바뀔 것으로 예상한다. 콘텐츠 변경 없는 etag 변경이나 그 반대는 위험 신호이다.

### 픽스처 관리

- 픽스처는 **최소한으로** 유지한다. 오류 픽스처는 하나의 실패를 보여 준다. `composition`은 빌드 언어를 검증하고 `basic`은 실제 앱과 같은 모습을 보여 주기 위해 존재한다. 경계 사례를 다루기 위해 `basic`을 키우지 말고 초점이 분명한 픽스처를 추가한다.
- 픽스처 주석도 문서이므로 다른 문서처럼 정확하게 유지한다.
- 스위트는 `../spec`을 읽는다. 픽스처는 형제 레이아웃을 전제로 하므로 세 하네스를 모두 갱신하지 않고 `spec/`을 이동하지 않는다.

## 문서

- 단일 진실 공급원(SSOT)의 경계는 다음과 같다. 규범 동작은 `spec/SPEC.md`, 튜토리얼은 `docs/`, 언어별 명령은 포트 README, 전체 예제는 `spec/fixtures/`에 둔다. 그 밖의 문서에서는 내용을 다시 서술하지 않고 링크한다.
- 변경 사항이 `docs/` 또는 README에 설명된 동작을 바꾸면 같은 PR에서 해당 문서도 갱신한다.

## 커밋 / PR 요구 사항

- PR 하나에는 논리적 변경 하나만 담고, 계약 변경은 다섯 단계를 모두 포함한다.
- PR 설명에 실행한 스위트를 명시한다.
- 새 포트 기여를 환영한다. [`docs/porting.ko.md`](docs/porting.ko.md)를 따른다. 기준은 리뷰 취향이 아니라 적합성 스위트이다.
