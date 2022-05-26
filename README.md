## List citas for given user CURP/RFC and email

```
curl 'https://citas.sat.gob.mx/api/td-citas-custom?rfcCurp=CIXN750827HNEHXK03&email=nchistyakoff@gmail.com' \
  -H 'authority: citas.sat.gob.mx' \
  -H 'accept: application/json, text/plain, */*' \
  -H 'accept-language: en-US,en;q=0.9' \
  -H 'cookie: XSRF-TOKEN=238013bf-667b-4c19-b87b-41cd60dd988f; 55328d4d2089d20635a8a69fe3b09b46=1bb75227e80fbc8844d13f4489fea163; JSESSIONID=79e9655c-6195-4811-b4b9-be48fdcf9325' \
  -H 'referer: https://citas.sat.gob.mx/listaCita' \
  -H 'sec-fetch-dest: empty' \
  -H 'sec-fetch-mode: cors' \
  -H 'sec-fetch-site: same-origin' \
  -H 'sec-gpc: 1' \
  -H 'user-agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.67 Safari/537.36' \
  --compressed

{"id":4330822,"rfcCurp":"CIXN750827HNEHXK03","nombreRazonSocial":"Nikolay Chistyakov","listaCitas":[{"id":4330822,"numeroCita":49823038,"nombreRazonSocial":"Nikolay Chistyakov","servicio":"Inscripción en el RFC de Personas Físicas","adsc":"MST Playa del Carmen","fecha":"02/05/2022 10:15:00","estatus":"CANCELADA","direccion":"Av. 45 Norte s.n., mza. 12, lote 10 y 11, entre calles 20 y 22, Col. Gonzálo Guerrero, 77710, Playa del Carmen, Solidaridad, Quintana Roo.","idRequisito":null,"idServicio":262,"urlTeams":null,"tipoSolicitante":"CASO_ESPECIAL","uuidAwsObjeto":null,"nombreOriginalArchivo":null,"intentosCarga":null,"disableUploadFiles":true}]}%
```

## Cancell cita

```
curl 'https://citas.sat.gob.mx/api/td-citas/4330822' \
  -X 'DELETE' \
  -H 'authority: citas.sat.gob.mx' \
  -H 'accept: application/json, text/plain, */*' \
  -H 'accept-language: en-US,en;q=0.9' \
  -H 'cookie: XSRF-TOKEN=238013bf-667b-4c19-b87b-41cd60dd988f; 55328d4d2089d20635a8a69fe3b09b46=951c08729913cc82fcb4b99857f19834; JSESSIONID=a3f97b0a-8f58-44ca-a025-9c0a3fd21210' \
  -H 'origin: https://citas.sat.gob.mx' \
  -H 'referer: https://citas.sat.gob.mx/listaCita' \
  -H 'sec-fetch-dest: empty' \
  -H 'sec-fetch-mode: cors' \
  -H 'sec-fetch-site: same-origin' \
  -H 'sec-gpc: 1' \
  -H 'user-agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.64 Safari/537.36' \
  -H 'x-xsrf-token: 238013bf-667b-4c19-b87b-41cd60dd988f' \
```
