/* global base_date, get_week_no, get_girl_id, get_target_id, get_calendar_base_url, get_result, get_start_date, selected_list_url, get_shop_id, get_select_course_url, get_tel_target_origin, get_phone_number */

// let parentScrollTop = 0;

// jQuery の処理
$(document).ready(function () {

    //-------------------------------
    // グローバル変数
    //-------------------------------
    var thDTAddInt = 0;
    var thDAddInt = 0;
    var small_device_flg = false; // 320px以下の狭い画面フラグ
    var win = window;
    var start_label = 1200;
    var end_label = 1800;

    $("body").addClass("fixed");

    // 「プロフィール」リンクをセットし直す
    $('.profileLink').on("click", function () {
        event.preventDefault(); // 通常リンクを無効化
        var profHref = $(this).attr("href"); // 既存のURLを取得
        $(this).attr("data-link", "on"); // クリックされた要素をイベントリスナで判定できるようマーキング
        // 外部通信
        //parent.window.postMessage({"function": "setURLQuery", "location": profHref}, heaven_domain);
        parent.window.postMessage({ "function": "setURLQuery", "location": profHref }, '*');
    });

    // PC用のヘッダ固定化関数
    function fixedHeaderPc() {
        // ドキュメントの左上から表のヘッダまでの位置を取得
        var fixTop = $('#chart .table').offset().top;
        var scrollFlg = false;
        $(window).scroll(function () {
            // 画面のスクロール量がヘッダ位置を越えたらFIXする
            if ($(window).scrollTop() >= fixTop) {
                if (scrollFlg === false) {
                    var clone = $("#chart table").clone();
                    scrollFlg = true;
                    $(clone).insertAfter("#chart table");
                    // 元のテーブル
                    $("#chart table:eq(0)").css("z-index", 10);
                    $("#chart table:eq(0) tbody").css("z-index", 5000);

                    // クローンしたテーブル
                    $("#chart table:eq(1)").css("position", "fixed").css("top", 0).css("z-index", 10000);

                    // クローンテーブルのヘッダのみ、元のテーブルより上にする
                    $("#chart table:eq(1) tbody").css("display", "none");
                }
            } else {
                $("#chart table:eq(1)").remove();
                scrollFlg = false;
            }
        });
    }

    fixedHeaderPc();

    $("btn2").bind({
        touchstart: function () {
            $(this).removeClass().addClass("touchstart");
        }
    });
    $("btn2").bind({
        touchend: function () {
            $(this).removeClass().addClass("touchend");
        }
    });

    $(".menu_color").bind({
        touchstart: function () {
            $(this).removeClass().addClass("touchstart");
        }
    });
    $(".menu_color").bind({
        touchend: function () {
            $(this).removeClass().addClass("touchend");
        }
    });

    // 背景色の設定
    var shop_color = $('.deco-shop-headline').css('background-color');
    $('a.menu_color').css("color", shop_color);
    var shop_background = $('.deco-shop-headline-font').css('color');
    $('a.menu_color').css("background-color", shop_background);

    // 初期表示
    setWeek(get_week_no, base_date);
    // 戻る処理
    $('.booking-footer span.back-btn a').attr('href', '');
    // ボタンのリンク内容をURLパラメータより変更する
    // ページによってボタンのパラメータを変更する
    var calenderWeeks = Math.floor(net_reservation_display_days / 7);
    if (net_reservation_display_days % 7 != 0) {
        calenderWeeks = calenderWeeks + 1;
    }
    var calenderPrevWeeks = parseInt(get_week_no) - 1;
    var calenderNextWeeks = parseInt(get_week_no) + 1;

    if (calenderPrevWeeks === 0) {
        $('.radius-box span.prev-btn a').removeClass("deco-shop-link");
        $('.radius-box span.prev-btn').addClass("unselected-prev");
        $('.radius-box span.prev-btn').removeClass("prev-btn");
    } else {
        if ((get_girl_id !== "") && (get_girl_id === get_target_id)) {
            $('.radius-box span.prev-btn a').attr('href', get_calendar_base_url + '/' + calenderPrevWeeks + '/' + get_girl_id);
        } else {
            $('.radius-box span.prev-btn a').attr('href', get_calendar_base_url + '/' + calenderPrevWeeks + '/');
        }
    }
    if (calenderNextWeeks > calenderWeeks) {
        $('.radius-box span.next-btn a').removeClass("deco-shop-link");
        $('.radius-box span.next-btn').addClass("unselected-next");
        $('.radius-box span.next-btn').removeClass("next-btn");
    } else {
        if ((get_girl_id !== "") && (get_girl_id === get_target_id)) {
            $('.radius-box span.next-btn a').attr('href', get_calendar_base_url + '/' + calenderNextWeeks + '/' + get_girl_id);
        } else {
            $('.radius-box span.next-btn a').attr('href', get_calendar_base_url + '/' + calenderNextWeeks + '/');
        }
    }
    // コメント表示クリック処理
    $('.girl-comment-block').on('click', function () {
        $(this).find(".contents-mini").hide();
        $(this).find(".contents-all").show();
    });

    // 午後(12:00)クリック処理
    $(document).on('click', '.jscDayLink', function (e) {
        e.preventDefault();
        scrollToTarget($('#jscDayLink'));
    });

    // 夜(18:00)クリック処理
    $(document).on('click', '.jscNightLink', function (e) {
        e.preventDefault();
        scrollToTarget($('#jscNightLink'));
    });

    // 12：00、18：00移動処理
    function scrollToTarget($target) {
        //        var scrollTag = ( window.chrome || 'WebkitAppearance' in document.documentElement.style )? 'body' : 'html';
        var scroll_height = 0;
        //        if (navigator.userAgent.indexOf('iPhone') > 0) {
        //            scroll_height = 90;
        //            $('html').contents().animate({
        //                scrollTop : $target.offset().top - scroll_height
        //            }, 350);
        //            return;
        //        } else if (navigator.userAgent.indexOf('iPad') > 0) {
        //            scroll_height = 90;
        //            $('html').contents().animate({
        //                scrollTop : $target.offset().top - scroll_height
        //            }, 350);
        //            return;
        //        } else if (navigator.userAgent.indexOf('Android') > 0) {
        if (navigator.userAgent.indexOf('Android') > 0) {
            scroll_height = 40;
        } else {
            scroll_height = 50;
        }
        $("body, html").animate({
            scrollTop: $target.offset().top - scroll_height
        }, 350);
    }

    /****************************
     * JSON データから表を作成する
     ****************************/
    // 文字列をJSONとして解析
    var calenderDate = JSON.parse(get_result);
    var count_calendar = (calenderDate.commu_acp_status).length;

    // 一週間の情報も取得できているかのチェック
    if (count_calendar === 7) {
        // 空き情報取得 API を Ajax で取得する
        if (calenderDate.commu_acp_status) {
            setTableToJson(calenderDate, get_start_date);
            /*
             // Loadingイメージを消す
             setTimeout(function () {
             removeLoading();
             $("table tbody").show();

             }, 1500);
             */
            removeLoading();
            $("table tbody").show();
        } else {
            // Loadingイメージを消す
            removeLoading();
            $("table thead").remove();
            $(".chart").remove();
        }
    } else {
        // 一週間以下の場合
        // Loadingイメージを消す
        removeLoading();
        $("table thead").remove();
        $(".chart").remove();
    }

    /**************
     * デザイン設定
     **************/
    // 「○○:00-」を大きい文字にする
    $("#chart tbody th.daytime-child[data-sys_time$=':00-']").each(function () {
        // 「○○:00-」を大きい文字にする
        $(this).css('font-size', '16px');
    });

    // 「○○:00-」を大きい文字にする
    $("#chart tbody th.daytime-child[data-sys_time$=':30-']").each(function () {
        // 「○○:00-」を大きい文字にする
        $(this).addClass('timeSharp');
    });

    // 横ラインの太さを設定する
    $("#chart tbody th.daytime-child[data-sys_time$=':00-'], th.daytime-child:eq(0)").each(function (i) {
        // 「○○:00-」行の上部を太ラインにする
        $(this).css('border-top-width', '2px');
        $(this).parent().find('td').css('border-top-width', '2px');

        // 営業開始時間が何時でも文字を大きくする
        if (i === 0) {
            $(this).css('font-size', '16px');
            $(this).removeClass('timeSharp');
        }
    });

    // 導線タブ切り替え（カレンダーから予約）
    $('#condition_calender').on("click", function (event) {
        event.preventDefault(); // 通常リンクを無効化
        $.ajax({
            type: "POST",
            url: selected_condition_url,
            data: {
                mode: '1'
            },
            success: function () {
                // 「カレンダーから予約」
                location.href = condition_calendar_url;
            }
        });
        return false;
    });
    // 導線タブ切り替え（コースから予約）
    $('#condition_course').on("click", function (event) {
        event.preventDefault(); // 通常リンクを無効化
        $.ajax({
            type: "POST",
            url: selected_condition_url,
            data: {
                mode: '2'
            },
            success: function () {
                // 「コースから予約」
                location.href = condition_course_url;
            }
        });
        return false;
    });
});
/*
 * ヘッダをfixedする
 */
function fixedTableHeader(small_device_flg, n, $thDayWidthArr, thDTAddInt, $thDaytimeWidth) {
    // テーブルの上端の位置を取得（216px)
    var anchor_top = $("#chart table").offset().top;

    // 親フレームがあれば、その位置を加算する
    var iframe_position = anchor_top;
    /*
     if($('iframe',parent.document)[0]){
     var iframe_position = $('iframe', parent.document).offset().top;
     anchor_top = anchor_top + iframe_position;
     }
     */

    // スクロールした量がテーブルの上端の位置を上回ったら、テーブルヘッダをクローンし
    // 画面上部にfixedする
    if ($(this).scrollTop() > (anchor_top)) {
        setTimeout(function () {
            if (n === 0) {
                $thDaytimeWidth = 0;
                if (small_device_flg !== true) {
                    $thDaytimeWidth = $('.table th.daytime').width() + thDTAddInt;
                } else {
                    $thDaytimeWidth = 90;
                }

                $('.table th.daytime').css({ width: $thDaytimeWidth }).css("top", iframe_position);
                var arrCnt = 0;
                $('.table th.day').each(function () {
                    var $th = $(this);
                    var cell_width;
                    if (small_device_flg !== true) {
                        cell_width = $thDayWidthArr[arrCnt];
                    } else {
                        cell_width = 34;
                    }
                    $th.css({ width: cell_width });
                    arrCnt++;
                });
                n = 1;
            }

            if ($('.header-hide').is(":hidden")) {
                //非表示だったら
                //fix型ヘッダを表示する
                $('.header-hide').css("display", 'inline');
                var day_title = $('thead').clone();
                $(".day_title").html(day_title);
                $(".day_title").addClass('fixed');
            }

            // Android 標準ブラウザの判定
            var ua = window.navigator.userAgent;
            if (/Android/.test(ua) && /Linux; U;/.test(ua) && !/Chrome/.test(ua)) {
                $(".day_title").css("left", "-1px");
            }
            var win = window;
            if (win.devicePixelRatio === 3) {
                $(".day_title").css("left", "-1px");
            }
        }, 600);
    }

    if ($(this).scrollTop() < (anchor_top)) {
        setTimeout(function () {
            $(".day_title").html('');
            $(".day_title").removeClass('fixed');
            $('.header-hide').css("display", 'none');
        }, 600);
    }
    return n;
}

/**
 * タイトル（yyyy年M日d日(基準日) - 終了日）を作成する
 *
 * @param d (日付のインスタンス)
 * @returns (yyyy年M日d日(基準日) - 終了日)
 */
function chart_title(d, present_week) {

    var selectweekdays = "<select id=\"selectCalendar\" onchange=\"getWeekData(this);\">";
    var calenderWeeks = Math.floor((net_reservation_display_days - 1) / 7) + 1;
    // 一週目用補正
    d.addDays(-1);
    for (var i = 1; i <= calenderWeeks; i++) {
        var baseDay = d.addDays(1).toString("MM月dd日");
        var endDay = d.addDays(6).toString("MM月dd日");
        selectweekdays = selectweekdays + "<option value=\"" + i + "\"";
        if (parseInt(present_week) === i) {
            selectweekdays = selectweekdays + "selected";
        }
        selectweekdays = selectweekdays + ">" + baseDay + "-" + endDay + "</option>";
    }
    selectweekdays = selectweekdays + "</select>";
    // 上記の処理でインスタンスの日付が進んでいるため巻き戻す
    d.addDays(-(calenderWeeks * 7 - 1));
    return selectweekdays;
}
function getWeekData(selectweek) {
    if ((get_girl_id !== "") && (get_girl_id === get_target_id)) {
        window.location.href = get_calendar_base_url + '/' + selectweek.value + '/' + get_girl_id;
    } else {
        window.location.href = get_calendar_base_url + '/' + selectweek.value + '/';
    }
}

/**
 * 表の見出しを作成する
 *
 * @param d (日付のインスタンス)
 */
function chart_table_title(d) {
    var cell_title;
    var dayOfWeekday;
    var dayOfWeek;
    var color;

    // タイトル加工時に、色も設定する様に修正
    dayOfWeekday = d.toString("d")
    if(mediaTypeName == "YOASOBI"){
      dayOfWeek = d.toString("ddd")
      cell_title = dayOfWeekday + "<br><span>" + dayOfWeek + "</span>";
    }else{
      dayOfWeek = dateNameConvert(d.toString("ddd"))
      cell_title = dayOfWeekday + "<br><span>(" + dayOfWeek + ")</span>";
    }
    color = setCellColor(d.toString("ddd"), d);
    $("#chart th.cell1").html(cell_title);
    $("#chart th.cell1").css('color', color);

    dayOfWeekday = d.addDays(1).toString("d")
    if(mediaTypeName == "YOASOBI"){
      dayOfWeek = d.toString("ddd")
      cell_title = dayOfWeekday + "<br><span>" + dayOfWeek + "</span>";
    }else{
      dayOfWeek = dateNameConvert(d.toString("ddd"))
      cell_title = dayOfWeekday + "<br><span>(" + dayOfWeek + ")</span>";
    }
    color = setCellColor(d.toString("ddd"), d);
    $("#chart th.cell2").html(cell_title);
    $("#chart th.cell2").css('color', color);

    dayOfWeekday = d.addDays(1).toString("d")
    if(mediaTypeName == "YOASOBI"){
      dayOfWeek = d.toString("ddd")
      cell_title = dayOfWeekday + "<br><span>" + dayOfWeek + "</span>";
    }else{
      dayOfWeek = dateNameConvert(d.toString("ddd"))
      cell_title = dayOfWeekday + "<br><span>(" + dayOfWeek + ")</span>";
    }
    color = setCellColor(d.toString("ddd"), d);
    $("#chart th.cell3").html(cell_title);
    $("#chart th.cell3").css('color', color);

    dayOfWeekday = d.addDays(1).toString("d")
    if(mediaTypeName == "YOASOBI"){
      dayOfWeek = d.toString("ddd")
      cell_title = dayOfWeekday + "<br><span>" + dayOfWeek + "</span>";
    }else{
      dayOfWeek = dateNameConvert(d.toString("ddd"))
      cell_title = dayOfWeekday + "<br><span>(" + dayOfWeek + ")</span>";
    }
    color = setCellColor(d.toString("ddd"), d);
    $("#chart th.cell4").html(cell_title);
    $("#chart th.cell4").css('color', color);

    dayOfWeekday = d.addDays(1).toString("d")
    if(mediaTypeName == "YOASOBI"){
      dayOfWeek = d.toString("ddd")
      cell_title = dayOfWeekday + "<br><span>" + dayOfWeek + "</span>";
    }else{
      dayOfWeek = dateNameConvert(d.toString("ddd"))
      cell_title = dayOfWeekday + "<br><span>(" + dayOfWeek + ")</span>";
    }
    color = setCellColor(d.toString("ddd"), d);
    $("#chart th.cell5").html(cell_title);
    $("#chart th.cell5").css('color', color);

    dayOfWeekday = d.addDays(1).toString("d")
    if(mediaTypeName == "YOASOBI"){
      dayOfWeek = d.toString("ddd")
      cell_title = dayOfWeekday + "<br><span>" + dayOfWeek + "</span>";
    }else{
      dayOfWeek = dateNameConvert(d.toString("ddd"))
      cell_title = dayOfWeekday + "<br><span>(" + dayOfWeek + ")</span>";
    }
    color = setCellColor(d.toString("ddd"), d);
    $("#chart th.cell6").html(cell_title);
    $("#chart th.cell6").css('color', color);

    dayOfWeekday = d.addDays(1).toString("d")
    if(mediaTypeName == "YOASOBI"){
      dayOfWeek = d.toString("ddd")
      cell_title = dayOfWeekday + "<br><span>" + dayOfWeek + "</span>";
    }else{
      dayOfWeek = dateNameConvert(d.toString("ddd"))
      cell_title = dayOfWeekday + "<br><span>(" + dayOfWeek + ")</span>";
    }
    color = setCellColor(d.toString("ddd"), d);
    $("#chart th.cell7").html(cell_title);
    $("#chart th.cell7").css('color', color);
}

// タイトルと表の見出しを選択された週で作成する
function setWeek(present_week, date) {
    d = new XDate(date); // 現在日時
    $(".radius-box .select_week").html(chart_title(d, present_week));
    //カレンダー基準日の更新　weekNoのTOP日付に更新
    d.addDays((get_week_no - 1) * 7);
    chart_table_title(d);
}

// 曜日を英語表記から日本語表記に変換する
function dateNameConvert(name) {
    var jp_youbi;
    switch (name) {
        case "Mon":
            jp_youbi = "月";
            break;
        case "Tue":
            jp_youbi = "火";
            break;
        case "Wed":
            jp_youbi = "水";
            break;
        case "Thu":
            jp_youbi = "木";
            break;
        case "Fri":
            jp_youbi = "金";
            break;
        case "Sat":
            jp_youbi = "土";
            break;
        case "Sun":
            jp_youbi = "日";
            break;
        default:
            break;
    }
    return jp_youbi;
}

// 見出しの曜日の色を設定する
function setCellColor(day_name, date) {
    var color = [];

    var year = date.getFullYear().toString().padStart(4, '0');
    var month = (date.getMonth() + 1).toString().padStart(2, '0');
    var day = date.getDate().toString().padStart(2, '0');
    var dateString = '' + year + month + day;
    for (let holiday of holidays) {
        if (dateString == holiday) {
            color = "red";
            return color;
        }
    }
    switch (day_name) {
        case "Sat":
            color = "blue";
            break;
        case "Sun":
            color = "red";
            break;
        default:
            color = "#666666";
            break;
    }
    return color;
}

// セルの背景色を設定する
function setBackgroundColor(register_name) {
    var color;
    switch (register_name) {
        case "○":
            back = "#31B16C";
            font = "#fff";
            size = "22px";
            break;
        case "◎":
            back = "#31B16C";
            font = "#fff";
            size = "22px";
            break;
        case "△":
            back = "#31B16C";
            font = "#fff";
            size = "22px";
            break;
        case "TEL":
            back = "#E74C53";
            font = "#fff";
            //color = "#fff";
            size = "12px";
            break;
        case "○\n先行":
            back = "#006400";
            font = "#fff";
            size = "10px";
            break;
        case "◎\n先行":
            back = "#006400";
            font = "#fff";
            size = "10px";
            break;
        case "△\n先行":
            back = "#006400";
            font = "#fff";
            size = "10px";
            break;
        case "待":
            back = "#fff";
            font = "#31B16C";
            size = "20px";
            break;
        default:
            back = "#fff";
            font = "#999";
            size = "22px";
            break;
    }
    color = [back, font, size];
    return color;
}

// JSON データから列を作成する
function setTableToJson(json, start_date) {
    var number = 0;
    var tr_number = 0; // 行数

    $("#chart tr").each(function () {
        // 見出しを除いた次の行からはじめるので、最初の行をスキップする

        if (tr_number > 0) {
            number = parseInt(number, 10);
            d = new XDate(start_date); // 開始時間

            // セルに表記する文字列の取得
            var json_res0 = json.commu_acp_status[0][d.toString("yyyy-MM-dd")][number].acp_status_mark;
            var json_res1 = json.commu_acp_status[1][d.addDays(1).toString("yyyy-MM-dd")][number].acp_status_mark;
            var json_res2 = json.commu_acp_status[2][d.addDays(1).toString("yyyy-MM-dd")][number].acp_status_mark;
            var json_res3 = json.commu_acp_status[3][d.addDays(1).toString("yyyy-MM-dd")][number].acp_status_mark;
            var json_res4 = json.commu_acp_status[4][d.addDays(1).toString("yyyy-MM-dd")][number].acp_status_mark;
            var json_res5 = json.commu_acp_status[5][d.addDays(1).toString("yyyy-MM-dd")][number].acp_status_mark;
            var json_res6 = json.commu_acp_status[6][d.addDays(1).toString("yyyy-MM-dd")][number].acp_status_mark;

            // 開始時間の取得
            var have_start0 = json.commu_acp_status[0][d.addDays(-6).toString("yyyy-MM-dd")][number].have_start;
            var have_start1 = json.commu_acp_status[1][d.addDays(1).toString("yyyy-MM-dd")][number].have_start;
            var have_start2 = json.commu_acp_status[2][d.addDays(1).toString("yyyy-MM-dd")][number].have_start;
            var have_start3 = json.commu_acp_status[3][d.addDays(1).toString("yyyy-MM-dd")][number].have_start;
            var have_start4 = json.commu_acp_status[4][d.addDays(1).toString("yyyy-MM-dd")][number].have_start;
            var have_start5 = json.commu_acp_status[5][d.addDays(1).toString("yyyy-MM-dd")][number].have_start;
            var have_start6 = json.commu_acp_status[6][d.addDays(1).toString("yyyy-MM-dd")][number].have_start;

            // システム日付の取得
            var sys_date0 = json.commu_acp_status[0][d.addDays(-6).toString("yyyy-MM-dd")][number].date;
            var sys_date1 = json.commu_acp_status[1][d.addDays(1).toString("yyyy-MM-dd")][number].date;
            var sys_date2 = json.commu_acp_status[2][d.addDays(1).toString("yyyy-MM-dd")][number].date;
            var sys_date3 = json.commu_acp_status[3][d.addDays(1).toString("yyyy-MM-dd")][number].date;
            var sys_date4 = json.commu_acp_status[4][d.addDays(1).toString("yyyy-MM-dd")][number].date;
            var sys_date5 = json.commu_acp_status[5][d.addDays(1).toString("yyyy-MM-dd")][number].date;
            var sys_date6 = json.commu_acp_status[6][d.addDays(1).toString("yyyy-MM-dd")][number].date;

            // ○、×、TELボタン表示
            //            okLinkDisplay(0, json_res0, this, dateDisplay(json.commu_acp_status[0]),have_start0);
            //            okLinkDisplay(1, json_res1, this, dateDisplay(json.commu_acp_status[1]),have_start1);
            //            okLinkDisplay(2, json_res2, this, dateDisplay(json.commu_acp_status[2]),have_start2);
            //            okLinkDisplay(3, json_res3, this, dateDisplay(json.commu_acp_status[3]),have_start3);
            //            okLinkDisplay(4, json_res4, this, dateDisplay(json.commu_acp_status[4]),have_start4);
            //            okLinkDisplay(5, json_res5, this, dateDisplay(json.commu_acp_status[5]),have_start5);
            //            okLinkDisplay(6, json_res6, this, dateDisplay(json.commu_acp_status[6]),have_start6);
            okLinkDisplay(0, json_res0, this, sys_date0, have_start0);
            okLinkDisplay(1, json_res1, this, sys_date1, have_start1);
            okLinkDisplay(2, json_res2, this, sys_date2, have_start2);
            okLinkDisplay(3, json_res3, this, sys_date3, have_start3);
            okLinkDisplay(4, json_res4, this, sys_date4, have_start4);
            okLinkDisplay(5, json_res5, this, sys_date5, have_start5);
            okLinkDisplay(6, json_res6, this, sys_date6, have_start6);

            number = parseInt(number, 10) + 1;
        }
        tr_number = parseInt(tr_number, 10) + 1;
    });

    // 予約状況によって背景色を変更する
    $("#chart td").each(function () {
        var cellColor;
        if ($("span", this).attr('data-mark') === undefined) {
            cellColor = setBackgroundColor($(this).html());
        } else {
            cellColor = setBackgroundColor($("span", this).attr('data-mark'));
        }
        $(this).css('background-color', cellColor[0]);
        $(this).css('color', cellColor[1]);
        $(this).css('font-size', cellColor[2]);
    });

    // (TELの)セルが続いたら結合する(1列目は0始まりで指定する）
    /*
     for (var i = 6; i > -1; i--) {
     rowCat(i, "TEL");
     }
     */
    // 高速化対策
    rowCat(6, "－", number);
    rowCat(5, "－", number);
    rowCat(4, "－", number);
    rowCat(3, "－", number);
    rowCat(2, "－", number);
    rowCat(1, "－", number);
    rowCat(0, "－", number);

    // 高速化対策
    rowCat(6, "TEL", number);
    rowCat(5, "TEL", number);
    rowCat(4, "TEL", number);
    rowCat(3, "TEL", number);
    rowCat(2, "TEL", number);
    rowCat(1, "TEL", number);
    rowCat(0, "TEL", number);
}

// 日付を取得する
function dateDisplay(date) {
    var date_disp = '';
    $.each(date, function (key, value) {
        date_disp = key;
    });
    return date_disp;
}

// Loadingイメージ削除関数
function removeLoading() {
    $("body").removeClass("fixed");
    $(".loader").remove();
    $(".loader-space").remove();
}

// 指定した列に指定の単語が連続してあったら、セルを結合する
function rowCat(num, string, number) {
    $('.concat_table').each(function () {
        var cell_status = 0; // セルの状態(0は繰り返しがない状態とする）

        $(this).find('tr').each(function () {
            $cnt = parseInt(num, 10);

            // セルの値を取得する
            var cell_val = $("td:eq(" + $cnt + ")", this).html();

            // セルの値が指定した単語で、まだ繰り返しがないなら、このTDをオブジェクトに格納する
            if ((cell_val === string) && (cell_status === 0)) {
                $td_base = $("td:eq(" + $cnt + ")", this);
            }

            // セルが指定した単語だったら結合する回数を増やす
            if (cell_val === string) {
                cell_status = parseInt(cell_status, 10) + 1;
            } else {
                // 指定した単語以外なら、カウントをクリアする
                cell_status = 0;
            }

            // 連続している回数が2以上だったら rowspan 属性を付与し、回数を結合の数に設定する
            if (cell_status > 1) {
                $td_base.attr("rowSpan", cell_status);
                $td_base.attr("data-name", string);
                if (string === '－') {
                    if (number === cell_status) {
                        // 縦列すべて'－'
                        $td_base.attr("data-name_all", '－');
                        if (number >= min_box_net_reservation) {
                            // ネット予約表示日数を越えているメッセージを出力できる縦枠数
                            $td_base.attr("data-name_message", net_reserv_days_message);
                        } else {
                            $td_base.attr("data-name_message", '－');
                        }

                    }
                }
                $("td:eq(" + $cnt + ")", this).remove(); // 現在のTDは削除する
            } else if (cell_status === 1) {
                $td_base.attr("data-name", string);
                $td_base.attr("rowSpan-single", cell_status);
            }
        });

        switch (string) {
            case 'TEL':
                // TELの場合、リンクを設定
                // 端末情報取得
                var userAgent = window.navigator.userAgent.toLowerCase();
                // アイコンの文字位置調整
                // iPhone,iPad,Android時
                if (userAgent.indexOf('iphone') !== -1 || userAgent.indexOf('ipad') !== -1 || userAgent.indexOf('android') !== -1) {
                    // カーソル変更あり
                    $("tr", this).find('td[data-name="TEL"]').html('<span style="color: #fff !important; cursor: pointer;">' + string + '</span>');
                } else {
                    // カーソル変更なし
                    $("tr", this).find('td[data-name="TEL"]').html('<span style="color: #fff !important;">' + string + '</span>');
                }

                break;
            case '－':
                // '－'の場合
                $("tr", this).find('td[data-name="－"]').html('<span class="vertical_msg" style="background-color: rgb(255, 255, 255); color: rgb(153, 153, 153);">－</span>');
                break;
        }
    });
}

/* ページ読み込み後 **************/
$(window).load(function () {

    // ネット予約表示日数を超えている日のメッセージ枠の表示を整える
    var tdH = $('#chart td[data-name_all="－"]:first').innerHeight();          // tdの高さを取得
    if (tdH === null) {
        // ネット予約表示日数を超えている日がない場合は処理を行わない
        return;
    }

    // ネット予約表示日数を超えている日に文言表示する。
    var msg = $('#chart td[data-name_all="－"]:first').attr("data-name_message");
    $('#chart td[data-name_all="－"]:first span').text(msg);
    $('#chart td[data-name_all="－"]:gt(0) span').text(''); // 列結合する先頭以外のセルは文言を空にする

    count = 1;
    $('#chart td[data-name_all="－"]').each(function () {
        if ($("span", this).text() !== msg) {
            // ネット予約表示日数を超えている日が複数ある場合は列結合する。
            $('#chart td[data-name_all="－"]:first').attr("colSpan", ++count);
            $(this).remove(); // 現在のTDは削除する
        }
    });

    $('#chart td[data-name_all="－"] span').css('width', '1em');
    $('#chart td[data-name_all="－"] span').css('line-height', '1.1em');

});

// テーブルのセルをリセットする
function resetTableCell() {
    // TDを一度クリアする
    $('#chart td').remove();
    var max_tr_length = console.log($('#chart tr').length);
    // 各行ごとにデフォルト値をもつセルを復帰させる
    // 高速化対策
    for (var i = 1; i < max_tr_length; i++) {
        $('#chart tr:eq(' + i + ')').append('<td>×</td><td>×</td><td>×</td><td>×</td><td>×</td><td>×</td><td>×</td>');
    }
}

// ○、◎、△ボタン表示
function okLinkDisplay(num, json, _this, date_disp, have_start) {
    // 開始時間が入っていたら ○ にして、開始時間を追記する
    // （お店カレンダーでは開始時間を表示しない）
    if ((have_start === null) || (typeof have_start === 'undefined') || (have_start === '') || (get_girl_id === "")) {
        have_start = '';
    }

    if (json === "○" || json === "◎" || json === "△") {
        if ((free_reservation === '1') && (resv_condition === '2') && (get_girl_id === "")) {
            // コースから予約：お店カレンダー（フリー）の場合、女の子の空き人数にかかわらず、△～◎ を すべて”○”表記とする
            json = "○";
        }
        // 先行利用がONで有る事
        if (advance_reservation_use_flg === "1") {
            //(先行ー１)/7余り切り捨て+1 で先行の表示境界週
            var advanceWeek = Math.floor((advance_reservation_days - 1) / 7) + 1;
            //先行利用日が表示週以降か
            if (parseInt(get_week_no) > advanceWeek || ((parseInt(get_week_no) === advanceWeek) && (num >= (advance_reservation_days - 1) % 7))) {
                // 表示週の場合、週の何日目から先行かを判断する
                json = json + "\n先行";
            }
        }
        $('td:eq(' + num + ')', _this).html('<span style="cursor: pointer;" data-mark="' + json + '">' + json + '</span><input type="hidden" class="hid_date" value="' + date_disp + '">' + '<p style="font-size: 10px!important">' + have_start + '</p>');
    } else if (json === "待") {
        $('td:eq(' + num + ')', _this).html('<span style="cursor: pointer;" data-mark="' + json + '"><img style="width:22px;margin-top:8px;" src="/img/bell.svg"/></span><input type="hidden" class="hid_date" value="' + date_disp + '">' + '<p style="font-size: 10px!important">' + have_start + '</p>');
    } else {
        $('td:eq(' + num + ')', _this).html(json);
    }
}

// ○、◎、△ボタンを押下時の処理
//$(document).on('click', '#chart tr td', function () {
$('#chart tr td').on('click', function () {
    var $cell_mark = $('span',this).attr('data-mark');
    var $cell = $(this);
    var start_time = $('p', this).text();

    if ($cell_mark === "○" || $cell_mark === "△" || $cell_mark === "◎" || $cell_mark === "○\n先行" || $cell_mark === "△\n先行" || $cell_mark === "◎\n先行" || $cell_mark === "待") {
        //        var day_time = $cell.parent().children(':first').text();
        var day_time = $cell.parent().children(':first').attr('data-sys_time');
        var day_date = $cell.find('.hid_date').val();   // クリックされたボタンの日付（hidden項目）
        var d = new XDate(day_date);
        day_date = day_date + "(" + dateNameConvert(d.toString("ddd")) + ")";   // クリックしたカレンダーデータの日付から曜日を取得
        if (get_girl_id !== "") {
            // 女の子カレンダー
            day_time = start_time !== '' ? start_time : day_time;
        }
        day_time = day_time.replace(/～/g, '-'); // ～ を - に置換

        let waitlist_notification = '0';
        var location_url = get_select_course_url;
        if ($cell_mark === "待") {
            waitlist_notification = '1';
            location_url = get_waiting_list_url;
        }

        $.ajax({
            type: "POST",
            url: time_change_proposal_url,
            data: JSON.stringify({
                girl_id: get_girl_id,
                day: day_date,
                day_time: day_time
            }),
            contentType: 'application/json',
            success: function (ret) {
                if (ret.result) {
                    // 前25分枠に提案可能な時間がある場合

                    // 予約時間を変更するボタンのインナーテキストを設定
                    let select_change_label = ret.resultData.dispProposalDayTime;
                    select_change_label = select_change_label + ' に変更する';
                    if (ret.resultData.proposalMessage) {
                      select_change_label = select_change_label + '\n(' + ret.resultData.proposalMessage + ')';
                    }
                    document.getElementById("select_change").innerText = select_change_label;

                    // 予約時間を変更しないボタンのインナーテキストを設定
                    let select_no_change_label = '変更せず\n' + ret.resultData.dispSelectedDayTime + ' のまま予約を進める';
                    document.getElementById("select_no_change").innerText = select_no_change_label;

                    document.getElementById('proposalGirl').value = get_girl_id;
                    document.getElementById('proposalDay').value = ret.resultData.proposalDay + "(" + ret.resultData.proposalDayOfWeek + ")";
                    document.getElementById('proposalDayTime').value = ret.resultData.proposalDayTime;
                    document.getElementById('proposalDayOfWeek').value = ret.resultData.proposalDayOfWeek;
                    document.getElementById('selectedDay').value = day_date;
                    document.getElementById('selectedDayTime').value = day_time;
                    document.getElementById('selectedDayOfWeek').value = ret.resultData.selectedDayOfWeek;

                    // クリックされたセルを表示領域の中央にスクロール
                    $cell.get(0).scrollIntoView({behavior: 'auto', block: 'center'});

                    // 予約時間変更提案モーダルを表示
                    // parentScrollTop = $(window.parent).scrollTop();
                    $('[data-remodal-id=time_change_proposal_modal]').remodal().open();
                    $('#timeChangeProposalModal').css('top', '0px');
                    const t = $cell.offset().top - 50 - $('#timeChangeProposalModal').offset().top;
                    $('#timeChangeProposalModal').css('top', t + 'px');
                } else {
                    $.ajax({
                        type: "POST",
                        url: selected_list_url,
                        data: {
                            girl_id: get_girl_id,
                            day: day_date,
                            day_time: day_time,
                            waitlist_notification: waitlist_notification
                        },
                        success: function () {
                            location.href = location_url;
                        }
                    });
                }
            }
        });


    } else if ($cell_mark === "TEL") {
        // 端末情報取得
        var userAgent = window.navigator.userAgent.toLowerCase();
        // iPhone,iPad,Android時のみ
        if (userAgent.indexOf('iphone') !== -1 || userAgent.indexOf('ipad') !== -1 || userAgent.indexOf('android') !== -1) {
            if (hv !== 'n') {
                // TELボタン押下した時の処理
                //parent.window.postMessage({"shop": {"tel": get_phone_number, "shop_id": get_shop_id}}, get_tel_target_origin);
                parent.window.postMessage({ "shop": { "tel": get_phone_number, "shop_id": get_shop_id } }, '*');
            }
        }
    } else {
        return;
    }

    // 予約時間を変更するボタンを押下時
    $('#select_change').on('click', function() {
        let select_girl_id = document.getElementById('proposalGirl').value;
        let select_day = document.getElementById('proposalDay').value;
        let select_day_time = document.getElementById('proposalDayTime').value;
        let backup_day = document.getElementById('selectedDay').value;
        let backup_day_time = document.getElementById('selectedDayTime').value;
        let backup_day_of_week = document.getElementById('selectedDayOfWeek').value;

        $.ajax({
            type: "POST",
            url: selected_list_url,
            data: {
                girl_id: select_girl_id,
                day: select_day,
                day_time: select_day_time,
                waitlist_notification: '0',
                backup_day: backup_day,
                backup_day_time: backup_day_time,
                backup_day_of_week: backup_day_of_week
            },
            success: function () {
                // 予約時間変更提案モーダルを閉じる
                $('[data-remodal-id=time_change_proposal_modal]').remodal().close();
                location.href = location_url;
            }
        });
    });
    // 予約時間を変更しないボタンを押下時
    $('#select_no_change').on('click', function() {
        $.ajax({
            type: "POST",
            url: selected_list_url,
            data: {
                girl_id: get_girl_id,
                day: day_date,
                day_time: day_time,
                waitlist_notification: '0'
            },
            success: function () {
                // 予約時間変更提案モーダルを閉じる
                $('[data-remodal-id=time_change_proposal_modal]').remodal().close();
                location.href = location_url;
            }
        });
    });
});

// 他の女の子を選ぶボタンを押下時の処理
$(document).on('click', '.btn2', function () {
    $.ajax({
        type: "GET",
        url: selected_list_url,
        success: function () {
        }
    });
});

// 予約リクエストアイコンクリック
$('.reservation_request_img_evt').on('click', function () {
    // 予約リクエスト準備処理実行
    $.ajax({
        type: "POST",
        url: prepare_reservation_request_url,
        success: function () {
            // 予約リクエスト利用規約画面に遷移
            location.href = reservation_request_terms_controller_url;
        }
    });
});

// 非プレミアム会員向け予約リクエストアイコンクリック
$('.open_heaven_premium_landing_img_evt').on('click', function () {
	const url = heavenPremiumLandingUrl + "&window=y";
	let width = screen.availWidth * 0.3;
    if (width > 500) {
        width = 500;
    }
    let height = screen.availHeight * 0.8;
    if (height > 1720) {
        height = 1720;
    }
    const option = 'width=' + width + ',height=' + height + ',scrollbars=no,resizable=yes'
    window.open(url, url, option);
});

$(document).on('closed', '#timeChangeProposalModal', function (event) {
  // $(window.parent).scrollTop(parentScrollTop);
});

$(window).on('load', function () {
    // 女の子コメント表示対応（画面表示、リサイズ時）
    $('.girl-comment-block').each(function () {
        if ($(this).find(".girl-comment-text").length) {
            var $textObj = $(this).find(".girl-comment-text");
            var ret = isEllipsisActiveInner($textObj);   // コメントが省略されているかどうか
            if (ret) {
                // 省略されている
                if ($(this).find(".girl-comment-btn").length) {
                    // 「続きを読む」が表示されていない時は「続きを読む」を表示
                    $(this).find(".girl-comment-btn").text("続きを読む");
                }
            } else {
                // 省略されていない
                $(this).find(".contents-mini").hide();
                $(this).find(".contents-all").show();
            }
        }
    });
});
function isEllipsisActiveInner($jQueryObject) {
    return ($jQueryObject.innerWidth() < $jQueryObject[0].scrollWidth);
}


