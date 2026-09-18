# sdui-template-compiler (Python)

> English: [README.md](README.md)

[SDUI 템플릿 컴파일러](../README.ko.md)의 Python 포트이다. Python ≥ 3.11을 사용하며 런타임 의존성은 하나(`PyYAML`)이다. TypeScript 레퍼런스를 포팅했으며 [`../spec`](../spec)의 공유 적합성 스위트(conformance suite), 즉 드리프트 가드가 바이트 단위 동일성을 보장한다.

아직 PyPI에 게시되지 않았다. 소스에서 설치한다.

## 설치

```sh
pip install path/to/sdui-template-compiler/py-cplr
# or editable, for development:
pip install -e 'path/to/sdui-template-compiler/py-cplr[dev]'
```

## CLI

```sh
sdui-compile <sdui-root> --out <dir> [--pretty]
```

`<dir>/manifest.json`과 `<dir>/screens/<id>/<templateVersion>.json`을 기록한다. `--pretty`는 스크린 JSON의 가독성에만 영향을 주며 etag는 항상 압축 형식의 해시이다.

## 라이브러리

```python
from sdui_template_compiler import ScreenRegistry

# Compiles every screen under the root eagerly — raises on any template error.
registry = ScreenRegistry.build("path/to/sdui-root")

# Newest version threshold ≤ the client app version
# (selection rules: ../docs/integration.md).
composed = registry.resolve("home", "2.3.0")
if composed is not None:
    composed.template  # client-ready JSON tree (plain dict/list/scalars)
    composed.etag      # sha256[:16] of the compact serialization

registry.get("home", "2.3.0")   # just the template
registry.ids()                  # all screen ids, in discovery order
registry.versions_of("home")    # app-version thresholds, ascending
registry.module_of("home")      # screen.yaml manifest (versions, params)
```

오류는 적합성 명세에서 고정한 진단 부분 문자열을 메시지에 포함하는 예외로 발생한다. 빌드 또는 부팅 경계에서 처리한다.

이 포트에만 해당하는 참고 사항은 두 가지이다.

- **YAML 방언**: 로더는 YAML 1.2 코어 의미론에 맞게 조정되어 있다(`on`/`yes`는 문자열로 남고 `1.0.0`도 문자열로 남는다). PyYAML의 기본 1.1 리졸버를 의도적으로 재정의한다.
- **직렬화**: `json_compat`는 `JSON.stringify` 숫자 형식을 재현하여 압축 출력과 etag가 레퍼런스와 바이트 단위로 일치하게 한다.

## 테스트

```sh
python3 -m venv .venv
.venv/bin/pip install -e '.[dev]'
.venv/bin/python -m pytest        # conformance suite against ../spec
```
