# [Шаблон backend сервера на Golang — часть 4 (Kubernetes (Helm), Horizontal Autoscaler)](https://habr.com/ru/post/724198/)

![Схема развертывания в Kubernetes](https://github.com/romapres2010/goapp/raw/master/doc/diagram/APP%20-%20Kebernates.jpg)

Ссылка на [репозиторий проекта](https://github.com/romapres2010/goapp).

Шаблон goapp в репозитории полностью готов к развертыванию в Docker, Docker Compose, Kubernetes (kustomize), Kubernetes (helm).

Ссылки на предыдущие части:
- [Первая часть](https://habr.com/ru/post/492062/) посвящена HTTP серверу.
- [Вторая часть](https://habr.com/ru/post/500554/) посвящена прототипированию REST API.
- [Третья часть](https://habr.com/ru/post/716634/) посвящена развертыванию шаблона в Docker, Docker Compose, Kubernetes (kustomize).
- [Четвертая часть](https://habr.com/ru/post/724198/) посвящена развертыванию в Kubernetes с Helm chart и настройке Horizontal Autoscaler.
- [Пятая часть](https://habr.com/ru/post/720286/) посвящена оптимизации Worker pool и особенностям его работы в составе микросервиса, развернутого в Kubernetes.

<cut />

## Содержание
1.Kubernetes (kustomize) vs  Kubernetes (helm)

## 1. Kubernetes (kustomize) vs Kubernetes (helm)
В сети большое количество статей с обсуждением преимуществ и недостатков каждого из подходов. Для себя я выделил следующие причины, зачем переходить на Helm:
- kustomize не позволяет передать какие-либо ENV переменные и повлиять "снаружи" на YAML
- kustomize не позволяет  
