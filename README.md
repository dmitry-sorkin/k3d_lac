# K3D Linear advance calibration model generator

Простой генератор моделей для калибровки linear/pressure advance, написанный на golang. Хостится [>тут<](https://k3d.tech/calibrations/la/k3d_la.html).
Автор не является профессиональным программистом. Код ужасен. При чтении кода остерегайтесь психологических травм.

--------

## TODO

- Добавить возможность калибровать smooth_time в клиппере;
- Добавить настройку потока;
- Добавить возможность вписывать свой начальный и конечный G-код печати;
- Добавить кнопку сброса значений к стандартным;
- Сделать так, чтобы список проверяемых значений показывался не при генерации файла, а при изменении количества сегментов, начального и конечного значения проверки.
