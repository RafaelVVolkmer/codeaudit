<!--
SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
SPDX-License-Identifier: MIT
-->

<a name="readme-top"></a>

<div align="center">

[![License][license-shield]][license-url]
[![Stars][stars-shield]][stars-url]
[![Forks][forks-shield]][forks-url]
[![Issues][issues-shield]][issues-url]
[![LinkedIn][linkedin-shield]][linkedin-url]

</div>

# codeaudit

`codeaudit` is a lightweight static code quality analyzer written in Go, designed
with **Clean Architecture** and **Clean Code** principles.

It focuses on:

- Minimal external dependencies (primarily Go standard library).
- Parallel analysis using goroutines and worker pools.
- Extensible metric model for complexity, size, coupling, comments and Git history.
- A clear separation between parsing, metric computation and reporting.

> **Status:** Prototype / reference implementation.  
> **Currently supported languages:** **C** and **Go** (designed to be extensible to others).

---

## 🚀 Basic usage

Build and run:

```bash
make build  
./bin/codeaudit analyze .
```

Or directly with Go:

```bash
go run ./cmd/codeaudit analyze .
```

Then inspect the persisted report:

```bash
./bin/codeaudit report -format json .
```

Reports are stored at:

```bash
&lt;root&gt;/.codeaudit/report.json
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## 💻 Tech Stack and Environment

| **Category**                    | **Technologies and Tools** |
|---------------------------------|----------------------------|
| **Implementation Language**     | [![Go](https://img.shields.io/badge/Go-white?style=for-the-badge&logo=go&logoColor=00ADD8&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://go.dev) |
| **Analyzed Languages (initial)**| [![C](https://img.shields.io/badge/C-white?style=for-the-badge&logo=c&logoColor=%23A8B9CC&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://en.cppreference.com/w/c/language.html) [![Go](https://img.shields.io/badge/Go-white?style=for-the-badge&logo=go&logoColor=00ADD8&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://go.dev) |
| **Build / Tooling**            | [![Go Toolchain](https://img.shields.io/badge/go%20toolchain-white?style=for-the-badge&logo=go&logoColor=00ADD8&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://go.dev/doc/install) |
| **Version Control**            | [![Git](https://img.shields.io/badge/Git-white?style=for-the-badge&logo=git&logoColor=%23F05032&logoSize=32&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://git-scm.com) [![GitHub](https://img.shields.io/badge/GitHub-white?style=for-the-badge&logo=github&logoColor=%23181717&logoSize=32&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://github.com) |
| **Documentation**              | [![Markdown](https://img.shields.io/badge/Markdown-white.svg?style=for-the-badge&logo=markdown&logoColor=%23000000&logoSize=32&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://www.markdownguide.org) |
| **Support Tools**              | [![Docker](https://img.shields.io/badge/Docker-white?style=for-the-badge&logo=docker&logoColor=%232496ED&logoSize=32&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://www.docker.com) |
| **Operating System**           | [![Linux](https://img.shields.io/badge/Linux-white?style=for-the-badge&logo=linux&logoColor=000000&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://kernel.org) |
| **Editor / IDE**               | [![Neovim](https://img.shields.io/badge/Neovim-white?style=for-the-badge&logo=neovim&logoColor=%2357A143&logoSize=32&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://neovim.io) [![VS Code](https://img.shields.io/badge/VS%20Code-white?style=for-the-badge&logo=visualstudiocode&logoColor=007ACC&labelColor=rgba(0,0,0,0)&color=rgba(0,0,0,0))](https://code.visualstudio.com) |

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## 🖥️ CLI Preview

A quick glimpse of the `codeaudit` CLI in action:

<p align="center">
  <img src="./readme/code_audit.svg" alt="CodeAudit CLI overview" />
</p>

This preview shows the text-based report format, including:

- Project summary (files, functions, average and max CCN).
- Hotspots (complexity × churn).
- Per-function metrics table with CCN, cognitive complexity, NLOC, parameters and more.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## 📚 References

The metric design and quality model in `codeaudit` are inspired by classic work
on software metrics, static analysis and maintainable design:

| Title                                                                                   | Author / Year                          |
|-----------------------------------------------------------------------------------------|----------------------------------------|
| *A Complexity Measure*                                                                  | Thomas J. McCabe, 1976                 |
| *Elements of Software Science* (Halstead Metrics)                                       | Maurice H. Halstead, 1977              |
| *Software Metrics: A Rigorous and Practical Approach*                                   | Norman Fenton, Shari Lawrence Pfleeger, 1996/2014 |
| *Code Complete: A Practical Handbook of Software Construction*                         | Steve McConnell, 2nd ed., 2004         |
| *Clean Code: A Handbook of Agile Software Craftsmanship*                               | Robert C. Martin, 2008                 |
| *Object-Oriented Software Construction*                                                | Bertrand Meyer, 2nd ed., 1997          |
| *Measuring Software Quality*                                                           | Capers Jones, 1996                     |
| *Software Metrics: Establishing a Company-Wide Program*                                | Alain Abran, Jeff W. Suryn et al., 2010 |

These references guide:

- The choice of complexity metrics (McCabe, Halstead).
- Aggregated maintainability indicators and risk mapping.
- Design and refactoring strategies for highly coupled or complex modules.
- The philosophy of keeping metrics actionable and developer-friendly.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

[maintainability-shield]: https://qlty.sh/gh/RafaelVVolkmer/projects/codeaudit/badges/maintainability.svg?style=flat-square
[maintainability-url]:   https://qlty.sh/gh/RafaelVVolkmer/projects/codeaudit

[stars-shield]: https://img.shields.io/github/stars/RafaelVVolkmer/codeaudit.svg?style=flat-square
[stars-url]: https://github.com/RafaelVVolkmer/codeaudit/stargazers

[contributors-shield]: https://img.shields.io/github/contributors/RafaelVVolkmer/codeaudit.svg?style=flat-square
[contributors-url]: https://github.com/RafaelVVolkmer/codeaudit/graphs/contributors

[forks-shield]: https://img.shields.io/github/forks/RafaelVVolkmer/codeaudit.svg?style=flat-square
[forks-url]: https://github.com/RafaelVVolkmer/codeaudit/network/members

[issues-shield]: https://img.shields.io/github/issues/RafaelVVolkmer/codeaudit.svg?style=flat-square
[issues-url]: https://github.com/RafaelVVolkmer/codeaudit/issues

[license-shield]: https://img.shields.io/github/license/RafaelVVolkmer/codeaudit.svg?style=flat-square
[license-url]: https://github.com/RafaelVVolkmer/codeaudit/blob/main/LICENSE

[linkedin-shield]: https://img.shields.io/badge/-LinkedIn-black.svg?style=flat-square&logo=linkedin&colorB=555
[linkedin-url]: https://www.linkedin.com/in/rafaelvvolkmer
