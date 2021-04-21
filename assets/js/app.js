window.onload = function() {
    fetch('/stats')
        .then(response => response.json())
        .then(data => {
            // console.log(data)
            const xaxes_data = ['x'].concat(Object.keys(data.Hashrates).map(x => new Date(x * 1)))
            // c3.generate({
            //     bindto: '#chart_shares',
            //     data: {
            //         x: 'x',
            //         columns: [
            //             xaxes_data,
            //             ['Workers'].concat(Object.values(data.Workers)),
            //             ['Shares'].concat(Object.values(data.ValidShares)),
            //             ['RejectedShares'].concat(Object.values(data.RejectedShares)),
            //             ['InvalidShares'].concat(Object.values(data.InvalidShares))
            //         ],
            //         type: 'area-spline',
            //         types: {
            //             Workers: 'step',
            //             Shares: 'area-spline',
            //             RejectedShares: 'area-spline',
            //             InvalidShares: 'area-spline'
            //         },
            //         // groups: [
            //         //     ['Shares', 'RejectedShares', 'InvalidShares']
            //         // ],
            //         axes: {
            //             Shares: 'y2'
            //         }
            //     },
            //     axis: {
            //         x: {
            //             type: 'timeseries',
            //             tick: {
            //                 fit: false,
            //                 rotate: -45,
            //                 multiline: false,
            //                 format: '%Y-%m-%d %I:%M %p'
            //             }
            //         },
            //         y: {
            //             label: {
            //                 text: 'Worker(s)',
            //                 position: 'outer-middle'
            //             }
            //         },
            //         y2: {
            //             show: true,
            //             label: {
            //                 text: 'Shares',
            //                 position: 'outer-middle'
            //             }
            //         }
            //     }
            // })

            // var chart = c3.generate({
            //     bindto: '#chart',
            //     data: {
            //         x: 'x',
            //         columns: [
            //             xaxes_data,
            //             ['Hashrate'].concat(Object.values(data.Hashrates)),
            //             ['Power'].concat(Object.values(data.Power))
            //         ],
            //         type: 'spline',
            //         axes: {
            //             Hashrate: 'y',
            //             Power: 'y2',
            //             Workers: 'y3'
            //         }
            //     },
            //     axis: {
            //         x: {
            //             type: 'timeseries',
            //             tick: {
            //                 fit: false,
            //                 rotate: -45,
            //                 multiline: false,
            //                 format: '%Y-%m-%d %I:%M %p'
            //             }
            //         },
            //         y: {
            //             label: {
            //                 text: 'Hashrate',
            //                 position: 'outer-middle'
            //             },
            //             tick: {
            //                 format: function(d, e) {
            //                     return `${d.toFixed(2)} MH/s`
            //                 }
            //             }
            //         },
            //         y2: {
            //             show: true,
            //             label: {
            //                 text: 'Power',
            //                 position: 'outer-middle'
            //             },
            //             tick: {
            //                 format: function(d, e) {
            //                     return `${d.toFixed(2)} W`
            //                 }
            //             }
            //         }
            //     }
            // });
        });
}